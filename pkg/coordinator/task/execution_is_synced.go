package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionIsSynced struct {
	bundle *Bundle
	client *execution.Client
	log    logrus.FieldLogger
}

var _ Runnable = (*ExecutionIsSynced)(nil)

const (
	NameExecutionIsSynced = "execution_is_synced"
)

func NewExecutionIsSynced(ctx context.Context, bundle *Bundle) *ExecutionIsSynced {
	bundle.log = bundle.log.WithField("task", NameExecutionIsSynced)

	return &ExecutionIsSynced{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),
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

	if status.IsSyncing {
		return false, nil
	}

	if status.Percent() != 100 {
		return false, nil
	}

	// Double check we've got some blocks just in case the node has only just booted up
	// and is still searching for peers that know the canonical chain.
	blockNumber, err := c.client.EthClient().BlockNumber(ctx)
	if err != nil {
		return false, nil
	}

	if blockNumber < 5 {
		return false, nil
	}

	return true, nil
}
