package types

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

type Coordinator interface {
	Logger() logrus.FieldLogger
	LogScope() *logger.LogScope
	ClientPool() *clients.ClientPool
	WalletManager() *wallet.Manager
	ValidatorNames() *names.ValidatorNames

	LoadTests(ctx context.Context)
	GetTestDescriptors() []TestDescriptor
	GetTestByRunID(runID uint64) Test
	GetTestQueue() []Test
	GetTestHistory() []Test
	ScheduleTest(descriptor TestDescriptor, configOverrides map[string]any, allowDuplicate bool) (Test, error)
}
