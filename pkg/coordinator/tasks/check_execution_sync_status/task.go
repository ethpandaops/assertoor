package checkexecutionsyncstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution/rpc"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_execution_sync_status"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks execution clients for their sync status.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx         *types.TaskContext
	options     *types.TaskOptions
	config      Config
	logger      logrus.FieldLogger
	firstHeight map[uint16]uint64
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:         ctx,
		options:     options,
		logger:      ctx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	t.processCheck(ctx)

	for {
		select {
		case <-time.After(t.config.PollInterval.Duration):
			t.processCheck(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *Task) processCheck(ctx context.Context) {
	allResultsPass := true
	failedClients := []string{}

	for _, client := range t.ctx.Scheduler.GetCoordinator().ClientPool().GetClientsByNamePatterns(t.config.ClientNamePatterns) {
		var checkResult bool

		checkLogger := t.logger.WithField("client", client.Config.Name)
		syncStatus, err := client.ExecutionClient.GetRPCClient().GetNodeSyncing(ctx)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			checkLogger.Warnf("errof fetching sync status: %v", err)

			checkResult = false
		} else {
			checkResult = t.processClientCheck(client, syncStatus, checkLogger)
		}

		if !checkResult {
			allResultsPass = false

			failedClients = append(failedClients, client.Config.Name)
		}
	}

	t.logger.Infof("Check result: %v, Failed Clients: %v", allResultsPass, failedClients)

	if allResultsPass {
		t.ctx.SetResult(types.TaskResultSuccess)
	} else {
		t.ctx.SetResult(types.TaskResultNone)
	}
}

func (t *Task) processClientCheck(client *clients.PoolClient, syncStatus *rpc.SyncStatus, checkLogger logrus.FieldLogger) bool {
	currentBlock, _ := client.ExecutionClient.GetLastHead()

	clientIdx := client.ExecutionClient.GetIndex()
	if t.firstHeight[clientIdx] == 0 {
		t.firstHeight[clientIdx] = currentBlock
	}

	if syncStatus.IsSyncing != t.config.ExpectSyncing {
		checkLogger.Debugf("check failed. check: ExpectSyncing, expected: %v, got: %v", t.config.ExpectSyncing, syncStatus.IsSyncing)
		return false
	}

	syncPercent := syncStatus.Percent()
	if syncPercent < t.config.ExpectMinPercent {
		checkLogger.Debugf("check failed. check: ExpectMinPercent, expected: >= %v, got: %v", t.config.ExpectMinPercent, syncPercent)
		return false
	}

	if syncPercent > t.config.ExpectMaxPercent {
		checkLogger.Debugf("check failed. check: ExpectMaxPercent, expected: <= %v, got: %v", t.config.ExpectMaxPercent, syncPercent)
		return false
	}

	if int64(currentBlock) < int64(t.config.MinBlockHeight) {
		checkLogger.Debugf("check failed. check: MinSlotHeight, expected: >= %v, got: %v", t.config.MinBlockHeight, currentBlock)
		return false
	}

	if t.config.WaitForChainProgression && currentBlock <= t.firstHeight[clientIdx] {
		checkLogger.Debugf("check failed. check: WaitForChainProgression, expected block height: >= %v, got: %v", t.firstHeight[clientIdx], currentBlock)
		return false
	}

	return true
}
