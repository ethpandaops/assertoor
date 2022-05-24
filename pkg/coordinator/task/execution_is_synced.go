package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionIsSyncedConfig struct {
	Percent                 float64 `yaml:"percent"`
	WaitForChainProgression bool    `yaml:"wait_for_chain_progression"`
	MinBlockHeight          int     `yaml:"min_block_height"`
}

type ExecutionIsSynced struct {
	bundle *Bundle
	client *execution.Client
	log    logrus.FieldLogger
	config ExecutionIsSyncedConfig
}

var _ Runnable = (*ExecutionIsSynced)(nil)

const (
	NameExecutionIsSynced = "execution_is_synced"
)

func NewExecutionIsSynced(ctx context.Context, bundle *Bundle, config ExecutionIsSyncedConfig) *ExecutionIsSynced {
	return &ExecutionIsSynced{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),
		log:    bundle.log.WithField("task", NameExecutionIsSynced),
	}
}

func DefaultExecutionIsSyncedConfig() ExecutionIsSyncedConfig {
	return ExecutionIsSyncedConfig{
		Percent:                 100,
		WaitForChainProgression: true,
		MinBlockHeight:          10,
	}
}

func (c *ExecutionIsSynced) Name() string {
	return NameExecutionIsSynced
}

func (c *ExecutionIsSynced) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ExecutionIsSynced) Start(ctx context.Context) error {
	return nil
}

func (c *ExecutionIsSynced) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ExecutionIsSynced) IsComplete(ctx context.Context) (bool, error) {
	status, err := c.client.SyncStatus(ctx)
	if err != nil {
		return false, err
	}

	c.log.WithField("percent", status.Percent()).Info("Sync status")

	if status.Percent() <= c.config.Percent {
		return false, nil
	}

	if !c.config.WaitForChainProgression {
		return true, nil
	}

	// Double check we've got some blocks just in case the node has only just booted up
	// and is still searching for peers that know the canonical chain.
	blockNumber, err := c.client.EthClient().BlockNumber(ctx)
	if err != nil {
		return false, nil
	}

	if blockNumber < uint64(c.config.MinBlockHeight) {
		return false, nil
	}

	return true, nil
}
