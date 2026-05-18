package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/logger"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/playbooklibrary"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/sirupsen/logrus"
)

type Coordinator interface {
	Logger() logrus.FieldLogger
	LogReader() logger.LogReader
	Database() *db.Database
	ClientPool() *clients.ClientPool
	WalletManager() *txmgr.Spamoor
	ValidatorNames() *names.ValidatorNames
	GlobalVariables() Variables
	TestRegistry() TestRegistry
	EventBus() *events.EventBus
	PlaybookLibrary() playbooklibrary.Service

	GetTestByRunID(runID uint64) Test
	GetTestQueue() []Test
	GetTestHistory(testID string, firstRunID uint64, offset uint64, limit uint64) ([]Test, uint64)

	// ScheduleTest preserves the original 4-arg signature for callers
	// outside this repo. Internally it forwards to
	// ScheduleTestWithOptions with SkipQueue/AllowDuplicate populated.
	ScheduleTest(descriptor TestDescriptor, configOverrides map[string]any, allowDuplicate bool, skipQueue bool) (TestRunner, error)

	// ScheduleTestWithOptions is the extended variant added for the
	// queue-picker UI. New code should prefer this; the simpler
	// signature is kept as a deprecated fallback.
	ScheduleTestWithOptions(descriptor TestDescriptor, configOverrides map[string]any, opts ScheduleOptions) (TestRunner, error)

	DeleteTestRun(runID uint64) error
}

// ScheduleOptions controls how a freshly scheduled test slots into
// the runner. AfterRunID > 0 places the new test immediately after
// that run in the pending queue; 0 falls back to either SkipQueue
// (off-queue, immediate parallel execution) or "append to the end"
// when SkipQueue is false.
type ScheduleOptions struct {
	AllowDuplicate bool
	SkipQueue      bool
	AfterRunID     uint64
}

type TestRegistry interface {
	AddLocalTest(testConfig *TestConfig) (TestDescriptor, error)
	AddLocalTestWithYaml(testConfig *TestConfig, yamlSource string) (TestDescriptor, error)
	AddExternalTest(ctx context.Context, extTestConfig *ExternalTestConfig) (TestDescriptor, error)
	DeleteTest(testID string) error
	GetTestDescriptors() []TestDescriptor

	// UpdateTestSchedule swaps the schedule (cron exprs + startup +
	// skipQueue) on a registered test and persists the change. Cron
	// expressions are validated before the in-memory state is touched.
	UpdateTestSchedule(testID string, schedule *TestSchedule) error
}
