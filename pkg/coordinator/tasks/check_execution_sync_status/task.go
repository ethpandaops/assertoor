package checkexecutionsyncstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
	"github.com/ethpandaops/minccino/pkg/coordinator/types"
	"github.com/imdario/mergo"
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
	config := DefaultConfig()
	if options.Config != nil {
		conf := &Config{}
		if err := options.Config.Unmarshal(&conf); err != nil {
			return nil, fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
		if err := mergo.Merge(&config, conf, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging task config for %v: %w", TaskName, err)
		}
	}
	return &Task{
		ctx:         ctx,
		options:     options,
		config:      config,
		logger:      ctx.Scheduler.GetLogger().WithField("task", TaskName),
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
	return t.options.Title
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

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}
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

func (t *Task) processCheck(ctx context.Context) error {
	allResultsPass := true
	failedClients := []string{}
	for _, client := range t.ctx.Scheduler.GetCoordinator().ClientPool().GetClientsByNamePatterns(t.config.ClientNamePatterns) {
		checkResult := t.processClientCheck(ctx, client)
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
	return nil
}

func (t *Task) processClientCheck(ctx context.Context, client *clients.PoolClient) bool {
	checkLogger := t.logger.WithField("client", client.Config.Name)
	syncStatus, err := client.ExecutionClient.GetRpcClient().GetNodeSyncing(ctx)
	if err != nil {
		checkLogger.Warnf("errof fetching sync status: %v", err)
		return false
	}
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
