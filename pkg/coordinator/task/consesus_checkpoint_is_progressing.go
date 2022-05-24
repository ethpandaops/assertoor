package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusCheckpointIsProgressing struct {
	bundle *Bundle
	client *consensus.Client
	log    logrus.FieldLogger

	checkpointName consensus.CheckpointName
	checkpoint     int64
}

var _ Runnable = (*ConsensusCheckpointIsProgressing)(nil)

const (
	NameConsensusCheckpointIsProgressing = "consensus_checkpoint_is_progressing"
)

func NewConsensusCheckpointIsProgressing(ctx context.Context, bundle *Bundle, checkpoint consensus.CheckpointName) *ConsensusCheckpointIsProgressing {
	return &ConsensusCheckpointIsProgressing{
		bundle: bundle,
		log:    bundle.log.WithField("task", NameConsensusCheckpointIsProgressing).WithField("checkpoint_name", checkpoint),

		checkpointName: checkpoint,
		checkpoint:     -1,
	}
}

func (c *ConsensusCheckpointIsProgressing) Name() string {
	return NameConsensusCheckpointIsProgressing
}

func (c *ConsensusCheckpointIsProgressing) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ConsensusCheckpointIsProgressing) Start(ctx context.Context) error {
	c.client = c.bundle.GetConsensusClient(ctx)

	return nil
}

func (c *ConsensusCheckpointIsProgressing) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ConsensusCheckpointIsProgressing) IsComplete(ctx context.Context) (bool, error) {
	if _, err := c.client.GetSpec(ctx); err != nil {
		return false, err
	}

	checkpoint, err := c.client.GetCheckpoint(ctx, c.checkpointName)
	if err != nil {
		return false, err
	}

	c.log.WithFields(logrus.Fields{
		"checkpoint":          checkpoint,
		"internal_checkpoint": c.checkpoint,
		"checkpoint_name":     c.checkpointName,
	}).Info("checking if checkpoint has progressed")

	// If the checkpoint is -1, we haven't gone through a cycle yet.
	if c.checkpoint == -1 {
		c.checkpoint = int64(checkpoint.Slot)

		return false, nil
	}

	// If the checkpoint has changed, we're still waiting.
	if c.checkpoint == int64(checkpoint.Slot) {
		return false, nil
	}

	return true, nil
}
