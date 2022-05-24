package task //nolint:dupl // false positive

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionIsUnhealthy struct {
	bundle *Bundle
	client *execution.Client
}

var _ Runnable = (*ExecutionIsUnhealthy)(nil)

const (
	NameExecutionIsUnhealthy = "execution_is_unhealthy"
)

func NewExecutionIsUnhealthy(ctx context.Context, bundle *Bundle) *ExecutionIsUnhealthy {
	bundle.log = bundle.log.WithField("task", NameExecutionIsUnhealthy)

	return &ExecutionIsUnhealthy{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),
	}
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
	return c.bundle.Logger()
}

func (c *ExecutionIsUnhealthy) IsComplete(ctx context.Context) (bool, error) {
	healthy, err := c.client.IsHealthy(ctx)
	if err != nil {
		return true, nil
	}

	return healthy, nil
}
