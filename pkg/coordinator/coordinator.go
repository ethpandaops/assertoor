package coordinator

import (
	"context"
	"fmt"

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
	}

	testToRun, err := test.CreateTest(ctx, bundle, c.Config.Test)
	if err != nil {
		return err
	}

	c.log.Info(fmt.Sprintf("starting test '%s' which contains %v tasks", testToRun.Name(), len(testToRun.Tasks())))

	if err := testToRun.Run(ctx); err != nil {
		return err
	}

	c.log.WithField("test", c.Config.Test).Info("test completed!")

	return nil
}
