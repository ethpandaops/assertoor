package runtaskmatrix

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

// stubTaskState reports a finished, successful child task so the watcher reads
// the task index map once and exits.
type stubTaskState struct {
	index types.TaskIndex
}

func (s *stubTaskState) Index() types.TaskIndex             { return s.index }
func (s *stubTaskState) ParentIndex() types.TaskIndex       { return 0 }
func (s *stubTaskState) ID() string                         { return "" }
func (s *stubTaskState) Name() string                       { return "stub" }
func (s *stubTaskState) Title() string                      { return "stub" }
func (s *stubTaskState) Description() string                { return "" }
func (s *stubTaskState) Config() any                        { return nil }
func (s *stubTaskState) Timeout() time.Duration             { return 0 }
func (s *stubTaskState) GetTaskStatusVars() types.Variables { return nil }
func (s *stubTaskState) GetScopeOwner() types.TaskIndex     { return 0 }

func (s *stubTaskState) GetTaskStatus() *types.TaskStatus {
	return &types.TaskStatus{Index: s.index, IsRunning: false, Result: types.TaskResultSuccess}
}

func (s *stubTaskState) GetTaskResultUpdateChan(oldResult types.TaskResult) <-chan bool {
	return nil
}

// stubScheduler runs the supplied watch function in its own goroutine, which is
// what concurrently reads the task index map.
type stubScheduler struct{}

func (s *stubScheduler) GetTaskState(taskIndex types.TaskIndex) types.TaskState {
	return &stubTaskState{index: taskIndex}
}
func (s *stubScheduler) GetTaskCount() uint64                   { return 0 }
func (s *stubScheduler) GetAllTasks() []types.TaskIndex         { return nil }
func (s *stubScheduler) GetRootTasks() []types.TaskIndex        { return nil }
func (s *stubScheduler) GetAllCleanupTasks() []types.TaskIndex  { return nil }
func (s *stubScheduler) GetRootCleanupTasks() []types.TaskIndex { return nil }
func (s *stubScheduler) GetServices() types.TaskServices        { return nil }
func (s *stubScheduler) GetTestRunID() uint64                   { return 0 }
func (s *stubScheduler) GetTestRunCtx() context.Context         { return context.Background() }
func (s *stubScheduler) TestResultPath() (string, error)        { return "", nil }

func (s *stubScheduler) ParseTaskOptions(_ helper.IRawMessage) (*types.TaskOptions, error) {
	return nil, nil
}

func (s *stubScheduler) ExecuteTask(ctx context.Context, taskIndex types.TaskIndex, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, taskIndex types.TaskIndex)) error {
	if taskWatchFn != nil {
		watchCtx, cancel := context.WithCancel(ctx)
		go taskWatchFn(watchCtx, cancel, taskIndex)
	}

	return nil
}

var (
	_ types.TaskSchedulerRunner = (*stubScheduler)(nil)
	_ types.TaskState           = (*stubTaskState)(nil)
)

// TestExecuteTaskIndexMapNoRace runs the matrix executor with many children so the
// per-child watcher goroutines read taskIdxMap while the executor spawns the
// remaining children. The index map must be fully written before any goroutine
// starts, otherwise this races (run with -race).
func TestExecuteTaskIndexMapNoRace(t *testing.T) {
	const childCount = 64

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logger.SetLevel(logrus.PanicLevel)

	tasks := make([]types.TaskIndex, childCount)
	for i := range tasks {
		tasks[i] = types.TaskIndex(i + 1)
	}

	config := DefaultConfig()
	config.RunConcurrent = true

	task := &Task{
		config:           config,
		logger:           logger,
		tasks:            tasks,
		taskIdxMap:       map[types.TaskIndex]int{},
		resultNotifyChan: make(chan taskResultUpdate, childCount),
	}
	task.ctx = &types.TaskContext{
		Scheduler:      &stubScheduler{},
		Index:          types.TaskIndex(1000),
		SetResult:      func(types.TaskResult) {},
		ReportProgress: func(float64, string) {},
	}

	done := make(chan error, 1)
	go func() {
		done <- task.Execute(context.Background())
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Execute did not complete in time")
	}

	if len(task.taskIdxMap) != childCount {
		t.Fatalf("taskIdxMap has %d entries, want %d", len(task.taskIdxMap), childCount)
	}
}
