package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/logger"
	"github.com/ethpandaops/assertoor/pkg/names"
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

	GetTestByRunID(runID uint64) Test
	GetTestQueue() []Test
	GetTestHistory(testID string, firstRunID uint64, offset uint64, limit uint64) ([]Test, uint64)
	ScheduleTest(descriptor TestDescriptor, configOverrides map[string]any, allowDuplicate bool, skipQueue bool) (TestRunner, error)
	DeleteTestRun(runID uint64) error
}

type TestRegistry interface {
	AddLocalTest(testConfig *TestConfig) (TestDescriptor, error)
	AddLocalTestWithYaml(testConfig *TestConfig, yamlSource string) (TestDescriptor, error)
	AddExternalTest(ctx context.Context, extTestConfig *ExternalTestConfig) (TestDescriptor, error)
	DeleteTest(testID string) error
	GetTestDescriptors() []TestDescriptor
}
