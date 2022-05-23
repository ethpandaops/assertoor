package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusIsSynced struct {
	bundle *Bundle
	client *consensus.Client
}

var _ Runnable = (*ConsensusIsSynced)(nil)

const (
	NameConsensusIsSynced = "consensus_is_synced"
)

func NewConsensusIsSynced(ctx context.Context, bundle *Bundle) *ConsensusIsSynced {
	bundle.log = bundle.log.WithField("task", NameConsensusIsSynced)

	return &ConsensusIsSynced{
		bundle: bundle,
	}
}

func (c *ConsensusIsSynced) Name() string {
	return NameConsensusIsSynced
}

func (c *ConsensusIsSynced) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ConsensusIsSynced) Start(ctx context.Context) error {
	c.client = c.bundle.GetConsensusClient(ctx)

	return nil
}

func (c *ConsensusIsSynced) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *ConsensusIsSynced) IsComplete(ctx context.Context) (bool, error) {
	status, err := c.client.GetSyncStatus(ctx)
	if err != nil {
		return false, err
	}

	c.bundle.log.WithField("percent", status.Percent()).Info("Sync status")

	if status.IsSyncing {
		return false, nil
	}

	// Check that our head slot is greater than 5 JUST in case of false positives.
	// Like if the node has just started up and hasn't started syncing yet.
	checkpoint, err := c.client.GetCheckpoint(ctx, consensus.Head)
	if err != nil {
		return false, err
	}

	return checkpoint.Slot > 5, nil
}
