package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusIsHealthyConfig struct {
}

type ConsensusIsHealthy struct {
	bundle *Bundle
	log    logrus.FieldLogger
	client *consensus.Client
	config ConsensusIsHealthyConfig
}

var _ Runnable = (*ConsensusIsHealthy)(nil)

const (
	NameConsensusIsHealthy = "consensus_is_healthy"
)

func NewConsensusIsHealthy(ctx context.Context, bundle *Bundle, config ConsensusIsHealthyConfig) *ConsensusIsHealthy {
	return &ConsensusIsHealthy{
		bundle: bundle,
		log:    bundle.log.WithField("task", NameConsensusIsHealthy),
		config: config,
	}
}

func DefaultConsensusIsHealthyConfig() ConsensusIsHealthyConfig {
	return ConsensusIsHealthyConfig{}
}

func (c *ConsensusIsHealthy) Name() string {
	return NameConsensusIsHealthy
}

func (c *ConsensusIsHealthy) Config() interface{} {
	return c.config
}

func (c *ConsensusIsHealthy) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *ConsensusIsHealthy) Start(ctx context.Context) error {
	c.client = c.bundle.GetConsensusClient(ctx)

	return nil
}

func (c *ConsensusIsHealthy) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ConsensusIsHealthy) IsComplete(ctx context.Context) (bool, error) {
	healthy, err := c.client.IsHealthy(ctx)
	if err != nil {
		return false, err
	}

	return healthy, nil
}
