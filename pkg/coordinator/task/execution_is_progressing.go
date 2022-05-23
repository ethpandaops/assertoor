package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionIsProgressing struct {
	bundle *Bundle
	client *execution.Client

	headBlockNumber uint64
}

var _ Runnable = (*ExecutionIsProgressing)(nil)

const (
	NameExecutionIsProgressing = "execution_is_progressing"
)

func NewExecutionIsProgressing(ctx context.Context, bundle *Bundle) *ExecutionIsProgressing {
	bundle.log = bundle.log.WithField("task", NameExecutionIsProgressing)

	return &ExecutionIsProgressing{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),

		headBlockNumber: 0,
	}
}

func (c *ExecutionIsProgressing) Name() string {
	return NameExecutionIsProgressing
}

func (c *ExecutionIsProgressing) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ExecutionIsProgressing) Start(ctx context.Context) error {
	return nil
}

func (c *ExecutionIsProgressing) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *ExecutionIsProgressing) IsComplete(ctx context.Context) (bool, error) {
	blockNumber, err := c.client.EthClient().BlockNumber(ctx)
	if err != nil {
		return false, err
	}

	// If the internal block number is not set, set it to the current block number,
	// and check again next time.
	if c.headBlockNumber == 0 {
		c.headBlockNumber = blockNumber
		return false, nil
	}

	// Check if the chain has progressed.
	if blockNumber > c.headBlockNumber {
		return true, nil
	}

	return false, nil
}
