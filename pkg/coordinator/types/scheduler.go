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
	ExecuteTask(ctx context.Context, taskIndex TaskIndex, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, taskIndex TaskIndex)) error
	WatchTaskPass(ctx context.Context, cancelFn context.CancelFunc, taskIndex TaskIndex)
	GetTaskState(taskIndex TaskIndex) TaskState
	GetTaskCount() int
	GetAllTasks() []TaskIndex
	GetRootTasks() []TaskIndex
	GetAllCleanupTasks() []TaskIndex
	GetRootCleanupTasks() []TaskIndex
}

type TaskServices interface {
	ClientPool() *clients.ClientPool
	WalletManager() *wallet.Manager
	ValidatorNames() *names.ValidatorNames
}
