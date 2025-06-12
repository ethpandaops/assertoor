package txpoollatencyanalysis

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

	load_tool := tx_load_tool.NewLoadTool(ctx, t.ctx, t.logger, t.wallet, client)

	// Generate and sending transactions, waiting for their propagation
	txs, latenciesMus, duplicatedP2PEventCount, coordinatedOmissionEventCount, notReceivedP2PEventCount, isFailed :=
		load_tool.ExecuteTPSLevel(t.config.TPS, t.config.Duration_s, testDeadline)

	totNumberOfTxes := len(txs)

	// Check if the context was cancelled or other errors occurred
	if ctx.Err() != nil && !isFailed {
		return nil
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

	// Calculate statistics
	var maxLatency int64 = 0
	var minLatency int64 = 0
	for _, lat := range latenciesMus {
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}
	t.logger.Infof("Max latency: %d mus, Min latency: %d mus", maxLatency, minLatency)

	// Generate HDR plot
	plot, err := hdr.HdrPlot(latenciesMus)
	if err != nil {
		t.logger.Errorf("Failed to generate HDR plot: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.ctx.Outputs.SetVar("tx_count", totNumberOfTxes)
	t.ctx.Outputs.SetVar("min_latency_mus", minLatency)
	t.ctx.Outputs.SetVar("max_latency_mus", maxLatency)
	t.ctx.Outputs.SetVar("duplicated_p2p_event_count", duplicatedP2PEventCount)
	t.ctx.Outputs.SetVar("missed_p2p_event_count", notReceivedP2PEventCount)
	t.ctx.Outputs.SetVar("coordinated_omission_event_count", coordinatedOmissionEventCount)

	t.ctx.SetResult(types.TaskResultSuccess)

	outputs := map[string]interface{}{
		"tx_count":                          totNumberOfTxes,
		"min_latency_mus":                   minLatency,
		"max_latency_mus":                   maxLatency,
		"tx_pool_latency_hdr_plot":          plot,
		"duplicated_p2p_event_count":        duplicatedP2PEventCount,
		"coordinated_omission_events_count": coordinatedOmissionEventCount,
	}

	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	return nil
}
