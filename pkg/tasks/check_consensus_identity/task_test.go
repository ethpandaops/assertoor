package checkconsensusidentity

import (
	"context"
	"io"
	"testing"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/logger"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

type fakeServices struct {
	pool *clients.ClientPool
}

func (f *fakeServices) Database() *db.Database                { return nil }
func (f *fakeServices) ClientPool() *clients.ClientPool       { return f.pool }
func (f *fakeServices) WalletManager() *txmgr.Spamoor         { return nil }
func (f *fakeServices) ValidatorNames() *names.ValidatorNames { return nil }
func (f *fakeServices) EventBus() *events.EventBus            { return nil }

type fakeSchedulerRunner struct {
	services types.TaskServices
}

func (f *fakeSchedulerRunner) GetServices() types.TaskServices { return f.services }
func (f *fakeSchedulerRunner) GetTestRunID() uint64            { return 1 }
func (f *fakeSchedulerRunner) GetTestRunCtx() context.Context  { return context.Background() }
func (f *fakeSchedulerRunner) ParseTaskOptions(_ helper.IRawMessage) (*types.TaskOptions, error) {
	return nil, nil
}
func (f *fakeSchedulerRunner) ExecuteTask(_ context.Context, _ types.TaskIndex, _ func(ctx context.Context, cancelFn context.CancelFunc, taskIndex types.TaskIndex)) error {
	return nil
}
func (f *fakeSchedulerRunner) TestResultPath() (string, error)                { return "", nil }
func (f *fakeSchedulerRunner) GetTaskState(_ types.TaskIndex) types.TaskState { return nil }
func (f *fakeSchedulerRunner) GetTaskCount() uint64                           { return 0 }
func (f *fakeSchedulerRunner) GetAllTasks() []types.TaskIndex                 { return nil }
func (f *fakeSchedulerRunner) GetRootTasks() []types.TaskIndex                { return nil }
func (f *fakeSchedulerRunner) GetAllCleanupTasks() []types.TaskIndex          { return nil }
func (f *fakeSchedulerRunner) GetRootCleanupTasks() []types.TaskIndex         { return nil }

func newTestTask(t *testing.T, cfg Config) *Task {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	pool, err := clients.NewClientPool(log)
	if err != nil {
		t.Fatalf("failed constructing client pool: %v", err)
	}

	services := &fakeServices{pool: pool}

	taskCtx := &types.TaskContext{
		Scheduler: &fakeSchedulerRunner{services: services},
		Vars:      vars.NewVariables(nil),
		Outputs:   vars.NewVariables(nil),
		Logger: logger.NewLogger(&logger.ScopeOptions{
			Parent: log,
		}),
		SetResult:      func(types.TaskResult) {},
		ReportProgress: func(float64, string) {},
	}

	return &Task{
		ctx:     taskCtx,
		options: &types.TaskOptions{},
		logger:  taskCtx.Logger.GetLogger(),
		config:  cfg,
	}
}

func TestProcessCheckNoMatchedClientsDoesNotPassAtDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ClientPattern = "does-not-match-anything"

	task := newTestTask(t, cfg)

	var gotResult types.TaskResult

	task.ctx.SetResult = func(r types.TaskResult) { gotResult = r }

	done, err := task.processCheck(1)
	if done {
		t.Fatalf("expected processCheck to keep waiting with zero matched clients, got done=true")
	}

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotResult == types.TaskResultSuccess {
		t.Fatalf("check reported Success with zero matched clients at default config")
	}
}

func TestProcessCheckNoMatchedClientsDoesNotPassWithZeroMinClientCount(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ClientPattern = "does-not-match-anything"
	cfg.MinClientCount = 0

	task := newTestTask(t, cfg)

	var gotResult types.TaskResult

	task.ctx.SetResult = func(r types.TaskResult) { gotResult = r }

	done, err := task.processCheck(1)
	if done {
		t.Fatalf("expected processCheck to keep waiting with zero matched clients, got done=true")
	}

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotResult == types.TaskResultSuccess {
		t.Fatalf("check reported Success with zero matched clients and minClientCount explicitly set to 0")
	}
}
