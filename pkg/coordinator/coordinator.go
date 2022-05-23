package coordinator

import (
	"context"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/test"
	"github.com/sirupsen/logrus"
)

type Coordinator struct {
	// Config is the coordinator configuration.
	Config *Config
	log    logrus.FieldLogger
}

func NewCoordinator(config *Config, log logrus.FieldLogger) *Coordinator {
	return &Coordinator{
		log:    log,
		Config: config,
	}
}

// Run executes the test until completion.
func (c *Coordinator) Run(ctx context.Context) error {
	c.log.WithField("config", c.Config).Info("starting coordinator")

	bundle := &test.Bundle{
		Log:          c.log,
		ConsensusURL: c.Config.Consensus.URL,
		ExecutionURL: c.Config.Execution.URL,
		TaskConfig:   c.Config.TaskConfig,
	}

	testToRun, err := test.NewTestByName(ctx, c.Config.Test, bundle)
	if err != nil {
		return err
	}

	if err := testToRun.Run(ctx); err != nil {
		return err
	}

	c.log.WithField("test", c.Config.Test).Info("test completed!")

	return nil
}
