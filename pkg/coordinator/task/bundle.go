package task

import "github.com/sirupsen/logrus"

// Bundle holds all possible configuration for a task.
type Bundle struct {
	log          logrus.FieldLogger
	ConsensusURL string
	ExecutionURL string
}

func NewBundle(log logrus.FieldLogger, consensusURL, executionURL string) *Bundle {
	return &Bundle{
		log:          log,
		ConsensusURL: consensusURL,
		ExecutionURL: executionURL,
	}
}

func (b *Bundle) Logger() logrus.FieldLogger {
	return b.log
}
