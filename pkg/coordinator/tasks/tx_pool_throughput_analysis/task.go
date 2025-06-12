package txpoolcheck

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/forkid"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/execution"
	"github.com/noku-team/assertoor/pkg/coordinator/helper"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/sentry"
	"github.com/noku-team/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "tx_pool_throughput_analysis"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the throughput of transactions in the Ethereum TxPool",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	wallet  *wallet.Wallet
	config  Config
	logger  logrus.FieldLogger
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	privKey, _ := crypto.HexToECDSA(config.PrivateKey)
	t.wallet, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	t.config = config
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	err := t.wallet.AwaitReady(ctx)
	if err != nil {
		return fmt.Errorf("cannot load wallet state: %w", err)
	}

	executionClients := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetReadyEndpoints(true)

	if len(executionClients) == 0 {
		t.logger.Errorf("No execution clients available")
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	client := executionClients[rand.Intn(len(executionClients))]

	conn, err := t.getTcpConn(ctx, client)
	if err != nil {
		t.logger.Errorf("Failed to get wire eth TCP connection: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	defer conn.Close()

	// Wait for the specified seconds before starting the task
	if t.config.SecondsBeforeRunning > 0 {
		t.logger.Infof("Waiting for %d seconds before starting the task...", t.config.SecondsBeforeRunning)
		select {
		case <-time.After(time.Duration(t.config.SecondsBeforeRunning) * time.Second):
			t.logger.Infof("Starting task after waiting.")
		case <-ctx.Done():
			t.logger.Warnf("Task cancelled before starting.")
			return ctx.Err()
		}
	}

	// Prepare to send transactions
	var totNumberOfTxes int = t.config.TPS * t.config.Duration_s
	var txs []*ethtypes.Transaction = make([]*ethtypes.Transaction, totNumberOfTxes)
	var testDeadline time.Time = time.Now().Add(time.Duration(t.config.Duration_s+60*30) * time.Second)

	startTime := time.Now()
	isFailed := false
	sentTxCount := 0
	coordinatedOmissionEventCount := 0

	// Start generating and sending transactions
	go func() {
		startExecTime := time.Now()
		endTime := startExecTime.Add(time.Second * time.Duration(t.config.Duration_s))

		// Generate and send transactions
		for i := 0; i < totNumberOfTxes; i++ {
			// Calculate how much time we have left
			remainingTime := time.Until(endTime)

			// Calculate sleep time to distribute remaining transactions evenly
			sleepTime := remainingTime / time.Duration(totNumberOfTxes-i)

			// generate and send tx
			go func(i int) {

				tx, err := t.generateTransaction(ctx, i)
				if err != nil {
					t.logger.Errorf("Failed to create transaction: %v", err)
					t.ctx.SetResult(types.TaskResultFailure)
					isFailed = true
					return
				}

				err = client.GetRPCClient().SendTransaction(ctx, tx)
				if err != nil {
					t.logger.WithField("client", client.GetName()).Errorf("Failed to send transaction: %v", err)
					t.ctx.SetResult(types.TaskResultFailure)
					isFailed = true
					return
				}

				txs[i] = tx
				sentTxCount++

				// log transaction sending
				if sentTxCount%t.config.MeasureInterval == 0 {
					elapsed := time.Since(startTime)
					t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
				}

			}(i)

			// Sleep to control the TPS
			if i < totNumberOfTxes-1 {
				if sleepTime > 0 {
					time.Sleep(sleepTime)
				} else {
					coordinatedOmissionEventCount++
				}
			}

			select {
			case <-ctx.Done():
				{
					t.logger.Warnf("Task cancelled, stopping transaction generation.")
					return
				}
			default:
				{
					// if testDeadline reached, stop sending txes
					if isFailed {
						return
					}
					if time.Now().After(testDeadline) {
						t.logger.Infof("Reached duration limit, stopping transaction generation.")
						return
					}
				}
			}
		}
	}()

	// Wait P2P event messages
	func() {
		var receivedEvents int = 0
		for {
			txes, err := conn.ReadTransactionMessages()
			if err != nil {
				t.logger.Errorf("Failed reading p2p events: %v", err)
				t.ctx.SetResult(types.TaskResultFailure)
				isFailed = true
				return
			}

			for _, tx := range *txes {
				tx_data := tx.Data()
				// read tx_data that is in the format "tx_index:<index>"
				var tx_index int
				_, err := fmt.Sscanf(string(tx_data), "tx_index:%d", &tx_index)
				if err != nil {
					t.logger.Errorf("Failed to parse transaction data: %v", err)
					t.ctx.SetResult(types.TaskResultFailure)
					isFailed = true
					return
				}
				if tx_index < 0 || tx_index >= totNumberOfTxes {
					t.logger.Errorf("Transaction index out of range: %d", tx_index)
					t.ctx.SetResult(types.TaskResultFailure)
					isFailed = true
					return
				}
				//latenciesMus[tx_index] = time.Since(txStartTime[tx_index]).Microseconds()
				receivedEvents++

				if receivedEvents%t.config.MeasureInterval == 0 {
					t.logger.Infof("Received %d p2p events", receivedEvents)
				}
			}

			if receivedEvents == totNumberOfTxes {
				t.logger.Infof("Reading of p2p events finished")
				return
			}

			select {
			case <-ctx.Done():
				t.logger.Warnf("Task cancelled, stopping reading p2p events.")
				return
			default:
				// check test deadline
				if time.Now().After(testDeadline) {
					t.logger.Warnf("Reached duration limit, stopping reading p2p events.")
					return
				}
			}
		}
	}()

	lastMeasureTime := time.Since(startTime)

	if coordinatedOmissionEventCount > 0 {
		t.logger.Warnf("Coordinated omission events count: %d", coordinatedOmissionEventCount)
	}

	// Send txes to other clients, for speeding up tx mining
	for _, tx := range txs {
		for _, otherClient := range executionClients {
			if otherClient.GetName() == client.GetName() {
				continue
			}

			otherClient.GetRPCClient().SendTransaction(ctx, tx)
		}
	}

	// Check if the context was cancelled or other errors occurred
	if ctx.Err() != nil && !isFailed {
		return nil
	}

	// Calculate statistics
	processed_tx_per_second := float64(sentTxCount) / lastMeasureTime.Seconds()

	t.ctx.Outputs.SetVar("mean_tps_throughput", processed_tx_per_second)
	t.logger.Infof("Processed %d transactions in %.2fs, mean throughput: %.2f tx/s", sentTxCount, lastMeasureTime.Seconds(), processed_tx_per_second)
	t.ctx.Outputs.SetVar("tx_count", totNumberOfTxes)
	t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, lastMeasureTime.Seconds())

	t.ctx.SetResult(types.TaskResultSuccess)

	outputs := map[string]interface{}{
		"tx_count":                          totNumberOfTxes,
		"mean_tps_throughput":               processed_tx_per_second,
		"coordinated_omission_events_count": coordinatedOmissionEventCount,
	}

	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	return nil
}

func (t *Task) getTcpConn(ctx context.Context, client *execution.Client) (*sentry.Conn, error) {
	chainConfig := params.AllDevChainProtocolChanges

	head, err := client.GetRPCClient().GetLatestBlock(ctx)
	if err != nil {
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	chainID, err := client.GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		return nil, err
	}

	chainConfig.ChainID = chainID

	genesis, err := client.GetRPCClient().GetEthClient().BlockByNumber(ctx, new(big.Int).SetUint64(0))
	if err != nil {
		t.logger.Errorf("Failed to fetch genesis block: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	conn, err := sentry.GetTcpConn(client)
	if err != nil {
		t.logger.Errorf("Failed to get TCP connection: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	forkId := forkid.NewID(chainConfig, genesis, head.NumberU64(), head.Time())

	// handshake
	err = conn.Peer(chainConfig.ChainID, genesis.Hash(), head.Hash(), forkId, nil)
	if err != nil {
		return nil, err
	}

	t.logger.Infof("Connected to %s", client.GetName())

	return conn, nil
}

func (t *Task) generateTransaction(ctx context.Context, i int) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		addr := t.wallet.GetAddress()
		toAddr := &addr

		txAmount, _ := crand.Int(crand.Reader, big.NewInt(0).SetUint64(10*1e18))

		feeCap := &helper.BigInt{Value: *big.NewInt(100000000000)} // 100 Gwei
		tipCap := &helper.BigInt{Value: *big.NewInt(1000000000)}   // 1 Gwei

		txObj := &ethtypes.DynamicFeeTx{
			ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: &tipCap.Value,
			GasFeeCap: &feeCap.Value,
			Gas:       50000,
			To:        toAddr,
			Value:     txAmount,
			Data:      []byte(fmt.Sprintf("tx_index:%d", i)),
		}

		return ethtypes.NewTx(txObj), nil
	})

	if err != nil {
		return nil, err
	}

	return tx, nil
}
