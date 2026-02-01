package checkexecutionsyncstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/execution/rpc"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_execution_sync_status"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks execution clients for their sync status.",
		Category:    "execution",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "goodClients",
				Type:        "array",
				Description: "Array of clients that meet sync criteria.",
			},
			{
				Name:        "failedClients",
				Type:        "array",
				Description: "Array of clients that do not meet sync criteria.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx         *types.TaskContext
	options     *types.TaskOptions
	config      Config
	logger      logrus.FieldLogger
	firstHeight map[uint16]uint64
}

type ClientInfo struct {
	Name          string `json:"name"`
	Synchronizing bool   `json:"synchronizing"`
	SyncHead      uint64 `json:"syncHead"`
	SyncDistance  uint64 `json:"syncDistance"`
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:         ctx,
		options:     options,
		logger:      ctx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
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
	checkCount := 0

	for {
		checkCount++

		if done := t.processCheck(ctx, checkCount); done {
			return nil
		}

		select {
		case <-time.After(t.config.PollInterval.Duration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) processCheck(ctx context.Context, checkCount int) bool {
	allResultsPass := true
	goodClients := []*ClientInfo{}
	failedClients := []*ClientInfo{}
	failedClientNames := []string{}

	for _, client := range t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "") {
		var checkResult bool

		checkLogger := t.logger.WithField("client", client.Config.Name)
		syncStatus, err := client.ExecutionClient.GetRPCClient().GetNodeSyncing(ctx)

		if ctx.Err() != nil {
			return false
		}

		if err != nil {
			checkLogger.Warnf("error fetching sync status: %v", err)

			checkResult = false
		} else {
			checkResult = t.processClientCheck(client, syncStatus, checkLogger)
		}

		if !checkResult {
			allResultsPass = false

			failedClients = append(failedClients, t.getClientInfo(client, syncStatus))
			failedClientNames = append(failedClientNames, client.Config.Name)
		} else {
			goodClients = append(goodClients, t.getClientInfo(client, syncStatus))
		}
	}

	t.logger.Infof("Check result: %v, Failed Clients: %v", allResultsPass, failedClientNames)

	if goodClientsData, err := vars.GeneralizeData(goodClients); err == nil {
		t.ctx.Outputs.SetVar("goodClients", goodClientsData)
	} else {
		t.logger.Warnf("Failed setting `goodClients` output: %v", err)
	}

	if failedClientsData, err := vars.GeneralizeData(failedClients); err == nil {
		t.ctx.Outputs.SetVar("failedClients", failedClientsData)
	} else {
		t.logger.Warnf("Failed setting `failedClients` output: %v", err)
	}

	if allResultsPass {
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, fmt.Sprintf("All clients synced: %d", len(goodClients)))

		return !t.config.ContinueOnPass
	}

	t.ctx.SetResult(types.TaskResultNone)
	t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for sync... %d/%d (attempt %d)", len(goodClients), len(goodClients)+len(failedClients), checkCount))

	return false
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

	if int64(currentBlock) < int64(t.config.MinBlockHeight) { //nolint:gosec // no overflow possible
		checkLogger.Debugf("check failed. check: MinBlockHeight, expected: >= %v, got: %v", t.config.MinBlockHeight, currentBlock)
		return false
	}

	if t.config.WaitForChainProgression && currentBlock <= t.firstHeight[clientIdx] {
		checkLogger.Debugf("check failed. check: WaitForChainProgression, expected block height: >= %v, got: %v", t.firstHeight[clientIdx], currentBlock)
		return false
	}

	return true
}

func (t *Task) getClientInfo(client *clients.PoolClient, syncStatus *rpc.SyncStatus) *ClientInfo {
	clientInfo := &ClientInfo{
		Name: client.Config.Name,
	}

	if syncStatus != nil {
		clientInfo.Synchronizing = syncStatus.IsSyncing
		clientInfo.SyncHead = syncStatus.CurrentBlock
		clientInfo.SyncDistance = syncStatus.HighestBlock - syncStatus.CurrentBlock
	}

	return clientInfo
}
