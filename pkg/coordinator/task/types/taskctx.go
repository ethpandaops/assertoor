package types

import (
	"context"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
	"github.com/ethpandaops/minccino/pkg/coordinator/helper"
	"github.com/sirupsen/logrus"
)

type TaskScheduler interface {
	GetLogger() logrus.FieldLogger
	GetClientPool() *clients.ClientPool
	ParseTaskOptions(rawtask *helper.RawMessage) (*TaskOptions, error)
	ExecuteTask(ctx context.Context, task Task, taskWatchFn func(task Task, ctx context.Context, cancelFn context.CancelFunc)) error
	WatchTaskPass(task Task, ctx context.Context, cancelFn context.CancelFunc)
	GetTaskStatus(task Task) *TaskStatus
	GetTaskResultUpdateChan(task Task, oldResult TaskResult) <-chan bool
}

type TaskStatus struct {
	IsStarted bool
	IsRunning bool
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
