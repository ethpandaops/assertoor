package checkconsensussyncstatus

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

// fakeServices satisfies types.TaskServices with a real, empty ClientPool
// and nil for everything else, since processCheck only touches ClientPool.
type fakeServices struct {
	pool *clients.ClientPool
}

func (f *fakeServices) Database() *db.Database                { return nil }
func (f *fakeServices) ClientPool() *clients.ClientPool       { return f.pool }
func (f *fakeServices) WalletManager() *txmgr.Spamoor         { return nil }
func (f *fakeServices) ValidatorNames() *names.ValidatorNames { return nil }
func (f *fakeServices) EventBus() *events.EventBus            { return nil }

// fakeSchedulerRunner satisfies types.TaskSchedulerRunner with only
// GetServices() wired to something real; nothing else here is reachable
// from processCheck.
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

func newTestTask(t *testing.T, clientPattern string) *Task {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	pool, err := clients.NewClientPool(log)
	if err != nil {
		t.Fatalf("failed constructing client pool: %v", err)
	}

	services := &fakeServices{pool: pool}
	taskVars := vars.NewVariables(nil)

	taskCtx := &types.TaskContext{
		Scheduler: &fakeSchedulerRunner{services: services},
		Index:     0,
		Vars:      taskVars,
		Outputs:   vars.NewVariables(nil),
		Logger: logger.NewLogger(&logger.ScopeOptions{
			Parent: log,
		}),
		SetResult:      func(types.TaskResult) {},
		ReportProgress: func(float64, string) {},
	}

	return &Task{
		ctx:         taskCtx,
		options:     &types.TaskOptions{},
		logger:      taskCtx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
		config: Config{
			ClientPattern: clientPattern,
		},
	}
}

func TestProcessCheckNoMatchedClientsDoesNotPass(t *testing.T) {
	task := newTestTask(t, "does-not-match-anything")

	var results []types.TaskResult

	task.ctx.SetResult = func(r types.TaskResult) {
		results = append(results, r)
	}

	for attempt := 1; attempt <= 3; attempt++ {
		done := task.processCheck(context.Background(), attempt)
		if done {
			t.Fatalf("attempt %d: processCheck reported done with zero matched clients", attempt)
		}
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 result updates, got %d", len(results))
	}

	for i, r := range results {
		if r == types.TaskResultSuccess {
			t.Fatalf("attempt %d: check reported Success with zero matched clients", i+1)
		}

		if r != types.TaskResultNone {
			t.Fatalf("attempt %d: expected TaskResultNone, got %v", i+1, r)
		}
	}
}
