package task //nolint:dupl // false positive

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusIsUnhealthy struct {
	bundle Bundle
	client *consensus.Client
}

var _ Runnable = (*ConsensusIsUnhealthy)(nil)

const (
	NameConsensusIsUnhealthy = "consensus_is_unhealthy"
)

func NewConsensusIsUnhealthy(ctx context.Context, bundle Bundle) *ConsensusIsUnhealthy {
	bundle.log = bundle.log.WithField("task", NameConsensusIsUnhealthy)

	return &ConsensusIsUnhealthy{
		bundle: bundle,
		client: bundle.GetConsensusClient(ctx),
	}
}

func (c *ConsensusIsUnhealthy) Name() string {
	return NameConsensusIsUnhealthy
}

func (c *ConsensusIsUnhealthy) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ConsensusIsUnhealthy) Start(ctx context.Context) error {
	return nil
}

func (c *ConsensusIsUnhealthy) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *ConsensusIsUnhealthy) IsComplete(ctx context.Context) (bool, error) {
	_, err := c.client.IsHealthy(ctx)
	if err != nil {
		return true, err
	}

	return false, nil
}
