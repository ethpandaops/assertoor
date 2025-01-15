package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/db"
	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
)

type TaskSchedulerRunner interface {
	TaskScheduler
	GetServices() TaskServices
	GetTestRunID() uint64
	ParseTaskOptions(rawtask helper.IRawMessage) (*TaskOptions, error)
	ExecuteTask(ctx context.Context, taskIndex TaskIndex, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, taskIndex TaskIndex)) error
	WatchTaskPass(ctx context.Context, cancelFn context.CancelFunc, taskIndex TaskIndex)
}

type TaskScheduler interface {
	GetTaskState(taskIndex TaskIndex) TaskState
	GetTaskCount() int
	GetAllTasks() []TaskIndex
	GetRootTasks() []TaskIndex
	GetAllCleanupTasks() []TaskIndex
	GetRootCleanupTasks() []TaskIndex
}

type TaskServices interface {
	Database() *db.Database
	ClientPool() *clients.ClientPool
	WalletManager() *wallet.Manager
	ValidatorNames() *names.ValidatorNames
}
