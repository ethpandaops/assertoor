package types

import (
	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
	"github.com/sirupsen/logrus"
)

type Coordinator interface {
	Logger() logrus.FieldLogger
	ClientPool() *clients.ClientPool
	GetTests() []Test
}
