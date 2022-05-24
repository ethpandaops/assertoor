package test

import (
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
	"github.com/sirupsen/logrus"
)

// Bundle holds the clients for a test.
type Bundle struct {
	Log          logrus.FieldLogger
	ConsensusURL string
	ExecutionURL string
	TaskConfig   task.Config
}

func (b *Bundle) AsTaskBundle() *task.Bundle {
	return task.NewBundle(b.Log, b.ConsensusURL, b.ExecutionURL, b.TaskConfig)
}
