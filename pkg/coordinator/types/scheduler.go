package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
)

type TaskScheduler interface {
	GetServices() TaskServices
	ParseTaskOptions(rawtask *helper.RawMessage) (*TaskOptions, error)
	ExecuteTask(ctx context.Context, task Task, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, task Task)) error
	WatchTaskPass(ctx context.Context, cancelFn context.CancelFunc, task Task)
	GetTaskCount() int
	GetAllTasks() []Task
	GetRootTasks() []Task
	GetAllCleanupTasks() []Task
	GetRootCleanupTasks() []Task
	GetTaskStatus(task Task) *TaskStatus
	GetTaskResultUpdateChan(task Task, oldResult TaskResult) <-chan bool
}

type TaskServices interface {
	ClientPool() *clients.ClientPool
	WalletManager() *wallet.Manager
	ValidatorNames() *names.ValidatorNames
}
