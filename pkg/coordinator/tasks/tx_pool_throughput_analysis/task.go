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

	var txs []*ethtypes.Transaction

	startTime := time.Now()
	sentTxCount := 0

	go func() {
		startExecTime := time.Now()
		endTime := startExecTime.Add(time.Second)

		for i := range t.config.QPS {
			// Calculate how much time we have left
			remainingTime := time.Until(endTime)

			// Calculate sleep time to distribute remaining transactions evenly
			sleepTime := remainingTime / time.Duration(t.config.QPS-i)

			// generate and sign tx
			go func() {
				tx, err := t.generateTransaction(ctx)
				if err != nil {
					t.logger.Errorf("Failed to create transaction: %v", err)
					t.ctx.SetResult(types.TaskResultFailure)
					return
				}

				sentTxCount++

				if sentTxCount%t.config.MeasureInterval == 0 {
					elapsed := time.Since(startTime)
					t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
				}

				err = client.GetRPCClient().SendTransaction(ctx, tx)
				if err != nil {
					t.logger.WithField("client", client.GetName()).Errorf("Failed to send transaction: %v", err)
					t.ctx.SetResult(types.TaskResultFailure)
					return
				}

				txs = append(txs, tx)
			}()

			time.Sleep(sleepTime)
		}

		execTime := time.Since(startExecTime)
		t.logger.Infof("Time to generate %d transactions: %v", t.config.QPS, execTime)
	}()

	lastMeasureTime := time.Now()
	gotTx := 0

	for gotTx < t.config.QPS {
		_txs, err := conn.ReadTransactionMessages()
		if err != nil {
			t.logger.Errorf("Failed to read transaction messages: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		gotTx += len(*_txs)

		if gotTx%t.config.MeasureInterval != 0 || gotTx >= 9900 {
			continue
		}

		t.logger.Infof("Got %d transactions", gotTx)
		t.logger.Infof("Tx/s: (%d txs processed): %.2f / s \n", t.config.MeasureInterval, float64(t.config.MeasureInterval)*float64(time.Second)/float64(time.Since(lastMeasureTime)))

		lastMeasureTime = time.Now()
	}

	totalTime := time.Since(startTime)
	t.logger.Infof("Total time for %d transactions: %.2fs", sentTxCount, totalTime.Seconds())

	// send to other clients, for speeding up tx mining
	for _, tx := range txs {
		for _, otherClient := range executionClients {
			if otherClient.GetName() == client.GetName() {
				continue
			}

			client.GetRPCClient().SendTransaction(ctx, tx)
		}
	}

	outputs := map[string]interface{}{
		"total_time_mus": totalTime.Microseconds(),
	}
	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	t.ctx.Outputs.SetVar("total_time_mus", totalTime.Milliseconds())
	t.ctx.SetResult(types.TaskResultSuccess)

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

func (t *Task) generateTransaction(ctx context.Context) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		addr := t.wallet.GetAddress()
		toAddr := &addr

		txAmount, _ := crand.Int(crand.Reader, big.NewInt(0).SetUint64(10*1e18))

		feeCap := &helper.BigInt{Value: *big.NewInt(100000000000)} // 100 Gwei
		tipCap := &helper.BigInt{Value: *big.NewInt(1000000000)}   // 1 Gwei

		var txObj ethtypes.TxData

		txObj = &ethtypes.DynamicFeeTx{
			ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: &tipCap.Value,
			GasFeeCap: &feeCap.Value,
			Gas:       50000,
			To:        toAddr,
			Value:     txAmount,
			Data:      []byte{},
		}

		return ethtypes.NewTx(txObj), nil
	})

	if err != nil {
		return nil, err
	}

	return tx, nil
}
