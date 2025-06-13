package txpool_latency_analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/tx_load_tool"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/hdr"
	"github.com/noku-team/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "tx_pool_latency_analysis"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the TxPool transaction propagation latency",
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

	t.logger.Infof("Measuring TxPool transaction propagation *latency*")
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
	load := tx_load_tool.NewLoad(load_target, t.config.TPS, t.config.Duration_s, testDeadline, t.config.LogInterval)

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
	var maxLatency int64 = 0
	var minLatency int64 = 0
	for _, lat := range result.LatenciesMus {
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}
	t.logger.Infof("Max latency: %d mus, Min latency: %d mus", maxLatency, minLatency)

	// Generate HDR plot
	plot, err := hdr.HdrPlot(result.LatenciesMus)
	if err != nil {
		t.logger.Errorf("Failed to generate HDR plot: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.ctx.Outputs.SetVar("tx_count", result.TotalTxs)
	t.ctx.Outputs.SetVar("min_latency_mus", minLatency)
	t.ctx.Outputs.SetVar("max_latency_mus", maxLatency)
	t.ctx.Outputs.SetVar("duplicated_p2p_event_count", result.DuplicatedP2PEventCount)
	t.ctx.Outputs.SetVar("missed_p2p_event_count", result.NotReceivedP2PEventCount)
	t.ctx.Outputs.SetVar("coordinated_omission_event_count", result.CoordinatedOmissionEventCount)

	t.ctx.SetResult(types.TaskResultSuccess)

	outputs := map[string]interface{}{
		"tx_count":                          result.TotalTxs,
		"min_latency_mus":                   minLatency,
		"max_latency_mus":                   maxLatency,
		"tx_pool_latency_hdr_plot":          plot,
		"duplicated_p2p_event_count":        result.DuplicatedP2PEventCount,
		"coordinated_omission_events_count": result.CoordinatedOmissionEventCount,
		"missed_p2p_event_count":            result.NotReceivedP2PEventCount,
	}

	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	return nil
}
