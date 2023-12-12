package types

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/sirupsen/logrus"
)

type Coordinator interface {
	Logger() logrus.FieldLogger
	ClientPool() *clients.ClientPool
	ValidatorNames() *names.ValidatorNames
	NewVariables(parentScope Variables) Variables
	GetTests() []Test
}
