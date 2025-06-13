package txpoolcheck

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/tx_load_tool"
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
		Description: "Checks the TxPool transaction propagation throughput",
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

	t.logger.Infof("Measuring TxPool transaction propagation *throughput*")
	t.logger.Infof("Targeting client: %s, TPS: %d, Duration: %d seconds",
		client.GetName(), t.config.TPS, t.config.Duration_s)

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

	// Prepare to collect transaction latencies
	var testDeadline time.Time = time.Now().Add(time.Duration(t.config.Duration_s+60*30) * time.Second)

	load_target := tx_load_tool.NewLoadTarget(ctx, t.ctx, t.logger, t.wallet, client)
	load := tx_load_tool.NewLoad(load_target, t.config.TPS, t.config.Duration_s, testDeadline)

	// Generate and sending transactions, waiting for their propagation
	err = load.Execute()
	if err != nil {
		t.logger.Errorf("Error during transaction load execution: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return err
	}

	// Collect the transactions and their latencies
	result, err := load.MeasurePropagationLatencies()
	if err != nil {
		t.logger.Errorf("Error measuring transaction propagation latencies: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return err
	}

	// Check if the context was cancelled or other errors occurred
	if result.Failed {
		return fmt.Errorf("Error measuring transaction propagation latencies: load failed")
	}

	// Send txes to other clients, for speeding up tx mining
	t.logger.Infof("Sending %d transactions to other clients for mining", len(result.Txs))
	for _, tx := range result.Txs {
		for _, otherClient := range executionClients {
			if otherClient.GetName() == client.GetName() {
				continue
			}

			otherClient.GetRPCClient().SendTransaction(ctx, tx)
		}
	}
	t.logger.Infof("Total transactions sent: %d", result.TotalTxs)

	// Calculate statistics
	t.logger.Infof("Last measure delay since start time: %s", result.LastMeasureDelay)

	processed_tx_per_second := float64(result.TotalTxs) / result.LastMeasureDelay.Seconds()

	t.logger.Infof("Processed %d transactions in %.2fs, mean throughput: %.2f tx/s",
		result.TotalTxs, result.LastMeasureDelay.Seconds(), processed_tx_per_second)
	t.logger.Infof("Sent %d transactions in %.2fs", result.TotalTxs, result.LastMeasureDelay.Seconds())

	t.ctx.Outputs.SetVar("mean_tps_throughput", processed_tx_per_second)
	t.ctx.Outputs.SetVar("tx_count", result.TotalTxs)
	t.ctx.Outputs.SetVar("duplicated_p2p_event_count", result.DuplicatedP2PEventCount)
	t.ctx.Outputs.SetVar("missed_p2p_event_count", result.NotReceivedP2PEventCount)
	t.ctx.Outputs.SetVar("coordinated_omission_event_count", result.CoordinatedOmissionEventCount)

	t.ctx.SetResult(types.TaskResultSuccess)

	outputs := map[string]interface{}{
		"tx_count":                          result.TotalTxs,
		"mean_tps_throughput":               processed_tx_per_second,
		"duplicated_p2p_event_count":        result.DuplicatedP2PEventCount,
		"coordinated_omission_events_count": result.CoordinatedOmissionEventCount,
		"missed_p2p_event_count":            result.NotReceivedP2PEventCount,
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
