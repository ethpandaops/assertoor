package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type ConsensusIsSyncedConfig struct {
	Percent                 float64 `yaml:"percent"`
	WaitForChainProgression bool    `yaml:"wait_for_chain_progression"`
	MinSlotHeight           int     `yaml:"min_slot_height"`
}

type ConsensusIsSynced struct {
	bundle *Bundle
	client *consensus.Client
	log    logrus.FieldLogger
	config ConsensusIsSyncedConfig
}

var _ Runnable = (*ConsensusIsSynced)(nil)

const (
	NameConsensusIsSynced = "consensus_is_synced"
)

func NewConsensusIsSynced(ctx context.Context, bundle *Bundle, config ConsensusIsSyncedConfig) *ConsensusIsSynced {
	return &ConsensusIsSynced{
		bundle: bundle,
		log:    bundle.Logger().WithField("task", NameConsensusIsSynced),
		config: config,
	}
}

func DefaultConsensusIsSyncedConfig() ConsensusIsSyncedConfig {
	return ConsensusIsSyncedConfig{
		Percent:                 100,
		WaitForChainProgression: true,
		MinSlotHeight:           10,
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
	return c.log
}

func (c *ConsensusIsSynced) IsComplete(ctx context.Context) (bool, error) {
	status, err := c.client.GetSyncStatus(ctx)
	if err != nil {
		return false, err
	}

	c.log.WithField("percent", status.Percent()).Info("Sync status")

	if status.Percent() >= c.config.Percent {
		return true, nil
	}

	if !c.config.WaitForChainProgression {
		return true, nil
	}

	// Check that our head slot is greater than the min slot height just to be sure.
	// Like if the node has just started up and hasn't started syncing yet.
	checkpoint, err := c.client.GetCheckpoint(ctx, consensus.Head)
	if err != nil {
		return false, err
	}

	return int(checkpoint.Slot) > c.config.MinSlotHeight, nil
}
