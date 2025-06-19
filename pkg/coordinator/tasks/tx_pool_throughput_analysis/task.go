package txpoolthroughputanalysis

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	txloadtool "github.com/erigontech/assertoor/pkg/coordinator/utils/tx_load_tool"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/ethereum/go-ethereum/crypto"
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

	if validationErr := config.Validate(); validationErr != nil {
		return validationErr
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

	n, randErr := rand.Int(rand.Reader, big.NewInt(int64(len(executionClients))))
	if randErr != nil {
		return fmt.Errorf("failed to generate random number: %w", randErr)
	}

	client := executionClients[n.Int64()]

	t.logger.Infof("Measuring TxPool transaction propagation *throughput*")
	t.logger.Infof("Targeting client: %s, TPS: %d, Duration: %d seconds",
		client.GetName(), t.config.TPS, t.config.DurationS)

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
	testDeadline := time.Now().Add(time.Duration(t.config.DurationS+60*30) * time.Second)

	loadTarget := txloadtool.NewLoadTarget(ctx, t.ctx, t.logger, t.wallet, client)
	load := txloadtool.NewLoad(loadTarget, t.config.TPS, t.config.DurationS, testDeadline, t.config.LogInterval)

	// Generate and sending transactions, waiting for their propagation
	execErr := load.Execute()
	if execErr != nil {
		t.logger.Errorf("Error during transaction load execution: %v", execErr)
		t.ctx.SetResult(types.TaskResultFailure)

		return execErr
	}

	// Collect the transactions and their latencies
	result, measureErr := load.MeasurePropagationLatencies()
	if measureErr != nil {
		t.logger.Errorf("Error measuring transaction propagation latencies: %v", measureErr)
		t.ctx.SetResult(types.TaskResultFailure)

		return measureErr
	}

	// Check if the context was cancelled or other errors occurred
	if result.Failed {
		return fmt.Errorf("error measuring transaction propagation latencies: load failed")
	}

	// Send txes to other clients, for speeding up tx mining
	// t.logger.Infof("Sending %d transactions to other clients for mining", len(result.Txs))

	// for _, tx := range result.Txs {
	// 	for _, otherClient := range executionClients {
	// 		if otherClient.GetName() == client.GetName() {
	// 			continue
	// 		}

	// 		if sendErr := otherClient.GetRPCClient().SendTransaction(ctx, tx); sendErr != nil {
	// 			t.logger.Errorf("Failed to send transaction to other client: %v", sendErr)
	// 			t.ctx.SetResult(types.TaskResultFailure)

	// 			return sendErr
	// 		}
	// 	}
	// }

	t.logger.Infof("Total transactions sent: %d", result.TotalTxs)

	// Calculate statistics
	t.logger.Infof("Last measure delay since start time: %s", result.LastMeasureDelay)

	processedTxPerSecond := float64(result.TotalTxs) / result.LastMeasureDelay.Seconds()

	t.logger.Infof("Processed %d transactions in %.2fs, mean throughput: %.2f tx/s",
		result.TotalTxs, result.LastMeasureDelay.Seconds(), processedTxPerSecond)
	t.logger.Infof("Sent %d transactions in %.2fs", result.TotalTxs, result.LastMeasureDelay.Seconds())

	t.ctx.Outputs.SetVar("mean_tps_throughput", processedTxPerSecond)
	t.ctx.Outputs.SetVar("tx_count", result.TotalTxs)
	t.ctx.Outputs.SetVar("duplicated_p2p_event_count", result.DuplicatedP2PEventCount)
	t.ctx.Outputs.SetVar("missed_p2p_event_count", result.NotReceivedP2PEventCount)
	t.ctx.Outputs.SetVar("coordinated_omission_event_count", result.CoordinatedOmissionEventCount)

	t.ctx.SetResult(types.TaskResultSuccess)

	outputs := map[string]interface{}{
		"tx_count":                          result.TotalTxs,
		"mean_tps_throughput":               processedTxPerSecond,
		"duplicated_p2p_event_count":        result.DuplicatedP2PEventCount,
		"coordinated_omission_events_count": result.CoordinatedOmissionEventCount,
		"missed_p2p_event_count":            result.NotReceivedP2PEventCount,
	}

	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	return nil
}
