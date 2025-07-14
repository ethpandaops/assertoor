package txpoolthroughputanalysis

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
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

type ThroughoutMeasure struct {
	LoadTPS      int `json:"load_tps"`
	ProcessedTPS int `json:"processed_tps"`
}

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
	t.logger.Infof("Targeting client: %s, Starting TPS: %d, Ending TPS: %d, Increment TPS: %d, Duration: %d seconds",
		client.GetName(), t.config.StartingTPS, t.config.EndingTPS, t.config.IncrementTPS, t.config.DurationS)

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

	// Create a new load target for the transaction propagation measurement
	loadTarget := txloadtool.NewLoadTarget(ctx, t.ctx, t.logger, t.wallet, client)

	percentile := 0.99 // 0.95 should be enough, change in the future if needed
	singleMeasureDeadline := time.Now().Add(time.Duration(t.config.DurationS+60*30) * time.Second)

	// slice of pairs: sending tps, processed TPS values
	var throughoutMeasures []ThroughoutMeasure

	// Iterate over the TPS range and crate a plot processedTps vs sendingTps
	t.logger.Infof("Iterating over the TPS range, starting TPS: %d, ending TPS: %d, increment TPS: %d",
		t.config.StartingTPS, t.config.EndingTPS, t.config.IncrementTPS)

	for sendingTps := t.config.StartingTPS; sendingTps <= t.config.EndingTPS; sendingTps += t.config.IncrementTPS {
		// measure the throughput with the current sendingTps
		processedTps, err := t.measureTpsWithLoad(loadTarget, sendingTps, t.config.DurationS, singleMeasureDeadline, percentile)
		if err != nil {
			t.logger.Errorf("Error during throughput measurement with sendingTps=%d, duration=%d: %v", sendingTps, t.config.DurationS, err)
			t.ctx.SetResult(types.TaskResultFailure)

			return err
		}

		// add to throughoutMeasures
		throughoutMeasures = append(throughoutMeasures, ThroughoutMeasure{
			LoadTPS:      sendingTps,
			ProcessedTPS: processedTps,
		})
	}

	t.logger.Infof("Finished measuring throughput, collected %d measures", len(throughoutMeasures))

	// Set the throughput measures in the task context outputs
	// from this plot we can compute the Maximum Sustainable Throughput or Capacity limit
	t.ctx.Outputs.SetVar("throughput_measures", throughoutMeasures) // log coordinated_omission_event_count and missed_p2p_event_count?

	outputs := map[string]interface{}{
		"throughput_measures": throughoutMeasures,
	}

	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	// Set the task result to success
	t.ctx.SetResult(types.TaskResultSuccess)

	return nil
}

func (t *Task) measureTpsWithLoad(loadTarget *txloadtool.LoadTarget, sendingTps, durationS int,
	testDeadline time.Time, percentile float64) (int, error) {
	t.logger.Infof("Single measure of throughput, sending TPS: %d, duration: %d secs", sendingTps, durationS)

	// Prepare to collect transaction latencies
	load := txloadtool.NewLoad(loadTarget, sendingTps, durationS, testDeadline, t.config.LogInterval)

	// Generate and sending transactions, waiting for their propagation
	execErr := load.Execute()
	if execErr != nil {
		t.logger.Errorf("Error during transaction load execution: %v", execErr)
		t.ctx.SetResult(types.TaskResultFailure)

		return 0, execErr
	}

	// Collect the transactions and their latencies
	result, measureErr := load.MeasurePropagationLatencies()
	if measureErr != nil {
		t.logger.Errorf("Error measuring transaction propagation latencies: %v", measureErr)
		t.ctx.SetResult(types.TaskResultFailure)

		return 0, measureErr
	}

	// Check if the context was cancelled or other errors occurred
	if result.Failed {
		return 0, fmt.Errorf("error measuring transaction propagation latencies: load failed")
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

	if percentile != 0.99 {
		// Calculate the percentile of latencies using result.LatenciesMus
		// Not implemented yet
		notImpl := errors.New("percentile selection not implemented, use 0.99")
		return 0, notImpl
	}

	t.logger.Infof("Using 0.99 percentile for latency calculation")

	t.logger.Infof("Last measure delay since start time: %s", result.LastMeasureDelay)

	processedTpsF := float64(result.TotalTxs) / result.LastMeasureDelay.Seconds()
	processedTps := int(processedTpsF) // round

	t.logger.Infof("Processed %d transactions in %.2fs, mean throughput: %.2f tx/s",
		result.TotalTxs, result.LastMeasureDelay.Seconds(), processedTpsF)
	t.logger.Infof("Sent %d transactions in %.2fs", result.TotalTxs, result.LastMeasureDelay.Seconds())

	return processedTps, nil
}
