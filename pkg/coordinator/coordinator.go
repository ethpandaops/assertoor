package coordinator

import (
	"context"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/test"
	"github.com/sirupsen/logrus"
)

type Coordinator struct {
	// Config is the coordinator configuration.
	Config Config
	log    logrus.FieldLogger
}

func NewCoordinator(config Config, log logrus.FieldLogger) *Coordinator {
	return &Coordinator{
		log:    log,
		Config: config,
	}
}

// Run executes the test until completion.
func (c *Coordinator) Run(ctx context.Context) error {
	bundle := test.Bundle{
		Log:          c.log,
		ConsensusURL: c.Config.Consensus.URL,
		ExecutionURL: c.Config.Execution.URL,
	}

	testToRun, err := test.NewTestByName(ctx, c.Config.Test, bundle)
	if err != nil {
		return err
	}

	return testToRun.Run(ctx)
}
