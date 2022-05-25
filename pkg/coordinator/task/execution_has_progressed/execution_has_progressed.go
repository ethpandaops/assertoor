package executionhasprogressed

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type Task struct {
	client *execution.Client
	log    logrus.FieldLogger
	config Config

	initialBlockHeight uint64
}

const (
	Name        = "execution_has_progressed"
	Description = "Finishes when the execution client has progressed the chain."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, executionURL string, config Config) *Task {
	return &Task{
		client: execution.GetExecutionClient(ctx, log, executionURL),
		log:    log.WithField("task", Name),
		config: config,

		initialBlockHeight: 0,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *Task) PollingInterval() time.Duration {
	return time.Second * 5
}

func (t *Task) Start(ctx context.Context) error {
	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	blockNumber, err := t.client.EthClient().BlockNumber(ctx)
	if err != nil {
		return false, err
	}

	// If the internal block number is not set, set it to the current block number,
	// and check again next time.
	if t.initialBlockHeight == 0 {
		t.initialBlockHeight = blockNumber

		return false, nil
	}

	// Check if the chain has progressed.
	if blockNumber-t.initialBlockHeight > uint64(t.config.Distance) {
		return true, nil
	}

	return false, nil
}
