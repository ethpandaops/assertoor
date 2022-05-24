package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ExecutionHasProgressedConfig struct {
	Distance int64
}

type ExecutionHasProgressed struct {
	bundle *Bundle
	client *execution.Client
	log    logrus.FieldLogger
	config ExecutionHasProgressedConfig

	initialBlockHeight uint64
}

var _ Runnable = (*ExecutionHasProgressed)(nil)

const (
	NameExecutionHasProgressed = "execution_has_progressed"
)

func NewExecutionHasProgressed(ctx context.Context, bundle *Bundle, config ExecutionHasProgressedConfig) *ExecutionHasProgressed {
	return &ExecutionHasProgressed{
		bundle: bundle,
		client: bundle.GetExecutionClient(ctx),
		log:    bundle.log.WithField("task", NameExecutionHasProgressed),
		config: config,

		initialBlockHeight: 0,
	}
}

func DefaultExecutionHasProgressedConfig() ExecutionHasProgressedConfig {
	return ExecutionHasProgressedConfig{
		Distance: 3,
	}
}

func (c *ExecutionHasProgressed) Name() string {
	return NameExecutionHasProgressed
}

func (c *ExecutionHasProgressed) Config() interface{} {
	return c.config
}

func (c *ExecutionHasProgressed) PollingInterval() time.Duration {
	return time.Second * 5
}

func (c *ExecutionHasProgressed) Start(ctx context.Context) error {
	return nil
}

func (c *ExecutionHasProgressed) Logger() logrus.FieldLogger {
	return c.log
}

func (c *ExecutionHasProgressed) IsComplete(ctx context.Context) (bool, error) {
	blockNumber, err := c.client.EthClient().BlockNumber(ctx)
	if err != nil {
		return false, err
	}

	// If the internal block number is not set, set it to the current block number,
	// and check again next time.
	if c.initialBlockHeight == 0 {
		c.initialBlockHeight = blockNumber
		return false, nil
	}

	// Check if the chain has progressed.
	if blockNumber-c.initialBlockHeight > uint64(c.config.Distance) {
		return true, nil
	}

	return false, nil
}
