package checkconsensussyncstatus

import (
	"context"
	"fmt"
	"time"

	"github.com/noku-team/assertoor/pkg/coordinator/clients"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/consensus/rpc"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_sync_status"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks consensus clients for their sync status.",
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

type ClientInfo struct {
	Name          string `json:"name"`
	Optimistic    bool   `json:"optimistic"`
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
	goodClients := []*ClientInfo{}
	failedClients := []*ClientInfo{}
	failedClientNames := []string{}

	for _, client := range t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "") {
		var checkResult bool

		checkLogger := t.logger.WithField("client", client.Config.Name)
		syncStatus, err := client.ConsensusClient.GetRPCClient().GetNodeSyncStatus(ctx)

		if ctx.Err() != nil {
			return
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
	} else {
		t.ctx.SetResult(types.TaskResultNone)
	}
}

func (t *Task) processClientCheck(client *clients.PoolClient, syncStatus *rpc.SyncStatus, checkLogger logrus.FieldLogger) bool {
	clientIdx := client.ExecutionClient.GetIndex()
	if t.firstHeight[clientIdx] == 0 {
		t.firstHeight[clientIdx] = syncStatus.HeadSlot
	}

	if syncStatus.IsSyncing != t.config.ExpectSyncing {
		checkLogger.Debugf("check failed. check: ExpectSyncing, expected: %v, got: %v", t.config.ExpectSyncing, syncStatus.IsSyncing)
		return false
	}

	if syncStatus.IsOptimistic != t.config.ExpectOptimistic {
		checkLogger.Debugf("check failed. check: ExpectOptimistic, expected: %v, got: %v", t.config.ExpectOptimistic, syncStatus.IsOptimistic)
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

	if int64(syncStatus.HeadSlot) < int64(t.config.MinSlotHeight) {
		checkLogger.Debugf("check failed. check: MinSlotHeight, expected: >= %v, got: %v", t.config.MinSlotHeight, syncStatus.HeadSlot)
		return false
	}

	if t.config.WaitForChainProgression && syncStatus.HeadSlot <= t.firstHeight[clientIdx] {
		checkLogger.Debugf("check failed. check: WaitForChainProgression, expected block height: >= %v, got: %v", t.firstHeight[clientIdx], syncStatus.HeadSlot)
		return false
	}

	return true
}

func (t *Task) getClientInfo(client *clients.PoolClient, syncStatus *rpc.SyncStatus) *ClientInfo {
	clientInfo := &ClientInfo{
		Name:          client.Config.Name,
		Synchronizing: syncStatus.IsSyncing,
		Optimistic:    syncStatus.IsOptimistic,
		SyncHead:      syncStatus.HeadSlot,
		SyncDistance:  syncStatus.SyncDistance,
	}

	return clientInfo
}
