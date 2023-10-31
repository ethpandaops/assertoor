package executionissynced

import (
	"context"
	"time"

	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type Task struct {
	executionURL string
	client       *execution.Client
	log          logrus.FieldLogger
	config       Config
	title        string
	timeout      time.Duration
}

const (
	Name        = "execution_is_synced"
	Description = "Waits until the execution client considers itself synced."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, executionURL string, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		executionURL: executionURL,
		client:       execution.GetExecutionClient(ctx, log, executionURL),
		log:          log.WithField("task", Name),
		config:       config,
		title:        title,
		timeout:      timeout,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Timeout() time.Duration {
	return t.timeout
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) Title() string {
	return t.title
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
	status, err := t.client.SyncStatus(ctx)
	if err != nil {
		return false, err
	}

	t.log.WithField("percent", status.Percent()).Info("Sync status")

	if status.Percent() < t.config.Percent {
		return false, nil
	}

	if !t.config.WaitForChainProgression {
		return true, nil
	}

	t.log.Info("Waiting for chain progression")

	// Double check we've got some blocks just in case the node has only just booted up
	// and is still searching for peers that know the canonical chain.
	blockNumber, err := t.client.EthClient().BlockNumber(ctx)
	if err != nil {
		return false, nil
	}

	t.log.WithField("block_number", blockNumber).WithField("required_height", t.config.MinBlockHeight).Info("sync status block number")

	if blockNumber < uint64(t.config.MinBlockHeight) {
		return false, nil
	}

	return true, nil
}
