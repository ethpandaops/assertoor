package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
)

type TaskSchedulerRunner interface {
	TaskScheduler
	GetServices() TaskServices
	GetTestRunID() uint64
	GetTestRunCtx() context.Context
	ParseTaskOptions(rawtask helper.IRawMessage) (*TaskOptions, error)
	ExecuteTask(ctx context.Context, taskIndex TaskIndex, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, taskIndex TaskIndex)) error

	// TestResultPath returns a filesystem path to a shared markdown file
	// that every task in this test run can write to. The file is created
	// lazily on first call. Whatever the tasks leave in it is persisted as
	// the test run's top-level result artifact, displayed by the UI as a
	// Result panel on the run page.
	TestResultPath() (string, error)
}

type TaskScheduler interface {
	GetTaskState(taskIndex TaskIndex) TaskState
	GetTaskCount() uint64
	GetAllTasks() []TaskIndex
	GetRootTasks() []TaskIndex
	GetAllCleanupTasks() []TaskIndex
	GetRootCleanupTasks() []TaskIndex
}

type TaskServices interface {
	Database() *db.Database
	ClientPool() *clients.ClientPool
	WalletManager() *txmgr.Spamoor
	ValidatorNames() *names.ValidatorNames
	EventBus() *events.EventBus
}
