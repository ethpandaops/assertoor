package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusIsUnhealthy struct {
	bundle *Bundle
	client *consensus.Client
}

var _ Runnable = (*ConsensusIsUnhealthy)(nil)

const (
	NameConsensusIsUnhealthy = "consensus_is_unhealthy"
)

func NewConsensusIsUnhealthy(ctx context.Context, bundle *Bundle) *ConsensusIsUnhealthy {
	bundle.log = bundle.log.WithField("task", NameConsensusIsUnhealthy)

	return &ConsensusIsUnhealthy{
		bundle: bundle,
	}
}

func (c *ConsensusIsUnhealthy) Name() string {
	return NameConsensusIsUnhealthy
}

func (c *ConsensusIsUnhealthy) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *ConsensusIsUnhealthy) Start(ctx context.Context) error {
	client := consensus.NewConsensusClient(c.bundle.log, c.bundle.ConsensusURL)
	if err := client.Bootstrap(ctx); err != nil {
		return nil
	}

	return nil
}

func (c *ConsensusIsUnhealthy) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *ConsensusIsUnhealthy) IsComplete(ctx context.Context) (bool, error) {
	if c.client == nil {
		return true, nil
	}

	_, err := c.client.IsHealthy(ctx)
	if err != nil {
		return true, err
	}

	return false, nil
}
