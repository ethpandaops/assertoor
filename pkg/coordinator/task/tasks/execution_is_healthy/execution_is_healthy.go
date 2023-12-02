package executionishealthy

import (
	"context"
	"time"

	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type Task struct {
	client  *execution.Client
	log     logrus.FieldLogger
	config  Config
	title   string
	timeout time.Duration
}

const (
	Name        = "execution_is_healthy"
	Description = "Performs a health check against the execution client."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, executionURL string, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		client:  execution.GetExecutionClient(ctx, log, executionURL),
		log:     log.WithField("task", Name),
		config:  config,
		title:   title,
		timeout: timeout,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) Title() string {
	return t.title
}

func (t *Task) Timeout() time.Duration {
	return t.timeout
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
	return time.Second * 1
}

func (t *Task) Start(ctx context.Context) error {
	return nil
}

func (t *Task) Cleanup(ctx context.Context) error {
	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	healthy, err := t.client.IsHealthy(ctx)
	if err != nil {
		return false, err
	}

	return healthy, nil
}
