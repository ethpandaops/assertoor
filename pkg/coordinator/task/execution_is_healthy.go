package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionIsHealthyConfig struct {
}

type ExecutionIsHealthy struct {
	bundle *Bundle
	client *execution.Client
	log    logrus.FieldLogger
	config ExecutionIsHealthyConfig
}

var _ Runnable = (*ExecutionIsHealthy)(nil)

const (
	NameExecutionIsHealthy = "execution_is_healthy"
)

func NewExecutionIsHealthy(ctx context.Context, bundle *Bundle, config ExecutionIsHealthyConfig) *ExecutionIsHealthy {
	return &ExecutionIsHealthy{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),
		log:    bundle.Logger().WithField("task", NameExecutionIsHealthy),
		config: config,
	}
}

func DefaultExecutionIsHealthyConfig() ExecutionIsHealthyConfig {
	return ExecutionIsHealthyConfig{}
}

func (c *ExecutionIsHealthy) Name() string {
	return NameExecutionIsHealthy
}

func (c *ExecutionIsHealthy) Config() interface{} {
	return c.config
}

func (c *ExecutionIsHealthy) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *ExecutionIsHealthy) Start(ctx context.Context) error {
	return nil
}

func (c *ExecutionIsHealthy) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ExecutionIsHealthy) IsComplete(ctx context.Context) (bool, error) {
	healthy, err := c.client.IsHealthy(ctx)
	if err != nil {
		return false, err
	}

	return healthy, nil
}
