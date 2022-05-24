package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusCheckpointHasProgressedConfig struct {
	Distance       int64                    `yaml:"distance"`
	CheckpointName consensus.CheckpointName `yaml:"checkpoint_name"`
}

type ConsensusCheckpointHasProgressed struct {
	bundle *Bundle
	client *consensus.Client
	log    logrus.FieldLogger
	config ConsensusCheckpointHasProgressedConfig

	checkpoint int64
}

var _ Runnable = (*ConsensusCheckpointHasProgressed)(nil)

const (
	NameConsensusCheckpointHasProgressed = "consensus_checkpoint_has_progressed"
)

func NewConsensusCheckpointHasProgressed(ctx context.Context, bundle *Bundle, config ConsensusCheckpointHasProgressedConfig) *ConsensusCheckpointHasProgressed {
	return &ConsensusCheckpointHasProgressed{
		bundle: bundle,
		log:    bundle.log.WithField("task", NameConsensusCheckpointHasProgressed).WithField("checkpoint_name", config.CheckpointName),
		config: config,

		checkpoint: -1,
	}
}

func DefaultConsensusCheckpointHasProgressed() ConsensusCheckpointHasProgressedConfig {
	return ConsensusCheckpointHasProgressedConfig{
		Distance:       3,
		CheckpointName: consensus.Head,
	}
}

func (c *ConsensusCheckpointHasProgressed) Name() string {
	return NameConsensusCheckpointHasProgressed
}

func (c *ConsensusCheckpointHasProgressed) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ConsensusCheckpointHasProgressed) Start(ctx context.Context) error {
	c.client = c.bundle.GetConsensusClient(ctx)

	return nil
}

func (c *ConsensusCheckpointHasProgressed) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ConsensusCheckpointHasProgressed) IsComplete(ctx context.Context) (bool, error) {
	if _, err := c.client.GetSpec(ctx); err != nil {
		return false, err
	}

	checkpoint, err := c.client.GetCheckpoint(ctx, c.config.CheckpointName)
	if err != nil {
		return false, err
	}

	c.log.WithFields(logrus.Fields{
		"checkpoint":          checkpoint,
		"internal_checkpoint": c.checkpoint,
	}).Info("checking if checkpoint has progressed")

	// If the checkpoint is -1, we haven't gone through a cycle yet.
	if c.checkpoint == -1 {
		c.checkpoint = int64(checkpoint.Slot)

		return false, nil
	}

	// If the checkpoint hasn't changed, we're still waiting.
	if c.checkpoint == int64(checkpoint.Slot) {
		return false, nil
	}

	if int64(checkpoint.Slot)-c.checkpoint >= c.config.Distance {
		return true, nil
	}

	return true, nil
}
