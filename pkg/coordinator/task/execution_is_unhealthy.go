package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionIsUnhealthyConfig struct {
}

type ExecutionIsUnhealthy struct {
	bundle *Bundle
	client *execution.Client
	log    logrus.FieldLogger
	config ExecutionIsUnhealthyConfig
}

var _ Runnable = (*ExecutionIsUnhealthy)(nil)

const (
	NameExecutionIsUnhealthy = "execution_is_unhealthy"
)

func NewExecutionIsUnhealthy(ctx context.Context, bundle *Bundle, config ExecutionIsUnhealthyConfig) *ExecutionIsUnhealthy {
	return &ExecutionIsUnhealthy{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),
		log:    bundle.log.WithField("task", NameExecutionIsUnhealthy),
		config: config,
	}
}

func DefaultExecutionIsUnhealthyConfig() ExecutionIsUnhealthyConfig {
	return ExecutionIsUnhealthyConfig{}
}

func (c *ExecutionIsUnhealthy) Name() string {
	return NameExecutionIsUnhealthy
}

func (c *ExecutionIsUnhealthy) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *ExecutionIsUnhealthy) Start(ctx context.Context) error {
	return nil
}

func (c *ExecutionIsUnhealthy) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ExecutionIsUnhealthy) IsComplete(ctx context.Context) (bool, error) {
	healthy, err := c.client.IsHealthy(ctx)
	if err != nil {
		return true, nil
	}

	return healthy, nil
}
