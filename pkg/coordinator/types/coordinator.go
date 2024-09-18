package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/db"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

type Coordinator interface {
	Logger() logrus.FieldLogger
	LogScope() *logger.LogScope
	Database() *db.Database
	ClientPool() *clients.ClientPool
	WalletManager() *wallet.Manager
	ValidatorNames() *names.ValidatorNames
	GlobalVariables() Variables

	AddLocalTest(testConfig *TestConfig) (TestDescriptor, error)
	AddExternalTest(ctx context.Context, extTestConfig *ExternalTestConfig) (TestDescriptor, error)
	GetTestDescriptors() []TestDescriptor
	GetTestByRunID(runID uint64) Test
	GetTestQueue() []Test
	GetTestHistory() []Test
	ScheduleTest(descriptor TestDescriptor, configOverrides map[string]any, allowDuplicate bool) (TestRunner, error)
}
