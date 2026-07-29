package scheduler

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/logger"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

type fakeServices struct{}

func (f *fakeServices) Database() *db.Database                { return nil }
func (f *fakeServices) ClientPool() *clients.ClientPool       { return nil }
func (f *fakeServices) WalletManager() *txmgr.Spamoor         { return nil }
func (f *fakeServices) ValidatorNames() *names.ValidatorNames { return nil }
func (f *fakeServices) EventBus() *events.EventBus            { return nil }

// blockingTask waits for its context to be cancelled, or gives up after a
// short deadline and reports that it ran to completion uninterrupted.
type blockingTask struct {
	wasCancelled    atomic.Bool
	ranToCompletion atomic.Bool
}

func (b *blockingTask) Config() interface{}    { return nil }
func (b *blockingTask) Timeout() time.Duration { return 0 }
func (b *blockingTask) LoadConfig() error      { return nil }

func (b *blockingTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		b.wasCancelled.Store(true)
		return ctx.Err()
	case <-time.After(150 * time.Millisecond):
		b.ranToCompletion.Store(true)
		return nil
	}
}

func addBlockingRootTask(ts *TaskScheduler) *blockingTask {
	task := &blockingTask{}
	descriptor := &types.TaskDescriptor{
		Name: "test-blocking-task",
		NewTask: func(_ *types.TaskContext, _ *types.TaskOptions) (types.Task, error) {
			return task, nil
		},
	}

	ts.taskStateMutex.Lock()
	ts.taskCount++
	idx := ts.taskCount

	state := &taskState{
		ts:             ts,
		index:          idx,
		options:        &types.TaskOptions{},
		descriptor:     descriptor,
		taskVars:       vars.NewVariables(nil),
		logger:         logger.NewLogger(&logger.ScopeOptions{Parent: ts.logger}),
		taskOutputs:    vars.NewVariables(nil),
		taskStatusVars: vars.NewVariables(nil),
	}

	ts.taskStateMap[idx] = state
	ts.allTasks = append(ts.allTasks, idx)
	ts.rootTasks = append(ts.rootTasks, idx)
	ts.taskStateMutex.Unlock()

	return task
}

func newTestScheduler() *TaskScheduler {
	log := logrus.New()
	log.SetOutput(io.Discard)

	return NewTaskScheduler(log, &fakeServices{}, vars.NewVariables(nil), 1)
}

// TestCancelTasksBeforeRunTasksIsHonored covers the abort-endpoint race: an
// operator can request cancellation between a test being registered and its
// worker goroutine reaching RunTasks. Before the fix, CancelTasks found
// cancelTaskCtx still nil and silently did nothing, so every root task ran
// to completion despite the abort.
func TestCancelTasksBeforeRunTasksIsHonored(t *testing.T) {
	ts := newTestScheduler()
	task := addBlockingRootTask(ts)

	ts.CancelTasks(true)

	if err := ts.RunTasks(context.Background(), 0); err == nil {
		t.Fatalf("expected RunTasks to return the cancellation error")
	}

	if !task.wasCancelled.Load() {
		t.Fatalf("expected the root task to observe cancellation requested before RunTasks started")
	}

	if task.ranToCompletion.Load() {
		t.Fatalf("expected the root task to be interrupted, not run to completion")
	}
}

// TestCancelTasksAfterRunTasksStartedIsHonored is the pre-existing case:
// once RunTasks has assigned cancelTaskCtx, CancelTasks correctly
// interrupts the running root task.
func TestCancelTasksAfterRunTasksStartedIsHonored(t *testing.T) {
	ts := newTestScheduler()
	task := addBlockingRootTask(ts)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = ts.RunTasks(context.Background(), 0)
	}()

	time.Sleep(30 * time.Millisecond)

	ts.CancelTasks(true)
	wg.Wait()

	if !task.wasCancelled.Load() {
		t.Fatalf("expected the root task to observe cancellation")
	}

	if task.ranToCompletion.Load() {
		t.Fatalf("expected the root task to be interrupted, not run to completion")
	}
}

// TestRunTasksNoAbortRunsNormally guards against the pending-cancel flag
// leaking into a run that was never aborted.
func TestRunTasksNoAbortRunsNormally(t *testing.T) {
	ts := newTestScheduler()
	task := addBlockingRootTask(ts)

	if err := ts.RunTasks(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if task.wasCancelled.Load() {
		t.Fatalf("task should not have been cancelled")
	}

	if !task.ranToCompletion.Load() {
		t.Fatalf("task should have run to completion")
	}
}

// TestCancelCtxFieldsNoRace exercises RunTasks, CancelTasks, and
// GetTestRunCtx concurrently under -race to guard the field-level locking.
func TestCancelCtxFieldsNoRace(t *testing.T) {
	ts := newTestScheduler()
	addBlockingRootTask(ts)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		_ = ts.RunTasks(context.Background(), 0)
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < 200; i++ {
			ts.CancelTasks(true)
			_ = ts.GetTestRunCtx()
		}
	}()

	wg.Wait()
}
