package types

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/sirupsen/logrus"
)

type Coordinator interface {
	Logger() logrus.FieldLogger
	ClientPool() *clients.ClientPool
	NewVariables(parentScope Variables) Variables
	GetTests() []Test
}
