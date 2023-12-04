package types

import (
	"context"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/helper"
	"github.com/sirupsen/logrus"
)

type TaskScheduler interface {
	GetLogger() logrus.FieldLogger
	GetCoordinator() Coordinator
	ParseTaskOptions(rawtask *helper.RawMessage) (*TaskOptions, error)
	ExecuteTask(ctx context.Context, task Task, taskWatchFn func(task Task, ctx context.Context, cancelFn context.CancelFunc)) error
	WatchTaskPass(task Task, ctx context.Context, cancelFn context.CancelFunc)
	GetTaskCount() int
	GetAllTasks() []Task
	GetRootTasks() []Task
	GetAllCleanupTasks() []Task
	GetRootCleanupTasks() []Task
	GetTaskStatus(task Task) *TaskStatus
	GetTaskResultUpdateChan(task Task, oldResult TaskResult) <-chan bool
}

type TaskStatus struct {
	Index     uint64
	IsStarted bool
	IsRunning bool
	StartTime time.Time
	StopTime  time.Time
	Result    TaskResult
	Error     error
}

type TaskContext struct {
	Scheduler  TaskScheduler
	Index      uint64
	ParentTask Task
	NewTask    func(options *TaskOptions) (Task, error)
	SetResult  func(result TaskResult)
}
