package types

import (
	"context"

	"github.com/sirupsen/logrus"
)

type TaskScheduler interface {
	ExecuteTask(ctx context.Context, task Task, taskWatchFn func(task Task, ctx context.Context, cancelFn context.CancelFunc)) error
	WatchTaskPass(task Task, ctx context.Context, cancelFn context.CancelFunc)
	GetTaskStatus(task Task) *TaskStatus
}

type TaskStatus struct {
	IsRunning bool
	Error     error
}

type TaskContext struct {
	Scheduler  TaskScheduler
	Index      uint64
	Logger     logrus.FieldLogger
	ParentTask Task
	NewTask    func(options *TaskOptions) (Task, error)
}
