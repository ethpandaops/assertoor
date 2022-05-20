package task //nolint:dupl // false positive

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusIsHealthy struct {
	bundle Bundle
	client *consensus.Client
}

var _ Runnable = (*ConsensusIsHealthy)(nil)

const (
	NameConsensusIsHealthy = "consensus_is_healthy"
)

func NewConsensusIsHealthy(ctx context.Context, bundle Bundle) *ConsensusIsHealthy {
	bundle.log = bundle.log.WithField("task", NameConsensusIsHealthy)

	return &ConsensusIsHealthy{
		bundle: bundle,
		client: bundle.GetConsensusClient(ctx),
	}
}

func (c *ConsensusIsHealthy) Name() string {
	return NameConsensusIsHealthy
}

func (c *ConsensusIsHealthy) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ConsensusIsHealthy) Start(ctx context.Context) error {
	return nil
}

func (c *ConsensusIsHealthy) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *ConsensusIsHealthy) IsComplete(ctx context.Context) (bool, error) {
	healthy, err := c.client.IsHealthy(ctx)
	if err != nil {
		return false, err
	}

	return healthy, nil
}
