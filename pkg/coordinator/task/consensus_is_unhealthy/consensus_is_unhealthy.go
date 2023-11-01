package consensusisunhealthy

import (
	"context"
	"time"

	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type Task struct {
	consensusURL string
	client       *consensus.Client
	log          logrus.FieldLogger
	config       Config
	title        string
	timeout      time.Duration

	done bool
}

const (
	Name        = "consensus_is_unhealthy"
	Description = "Performs a health check against the consensus client, finishes when the health checks fail."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, consensusURL string, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		consensusURL: consensusURL,
		log:          log.WithField("task", Name),
		config:       config,
		title:        title,
		timeout:      timeout,
		done:         false,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) Timeout() time.Duration {
	return t.timeout
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
	return time.Millisecond * 100
}

func (t *Task) Start(ctx context.Context) error {
	client := consensus.NewConsensusClient(t.log, t.consensusURL)

	client.Node().StartAsync(ctx)

	client.Node().OnHealthCheckFailed(ctx, func(ctx context.Context, event *beacon.HealthCheckFailedEvent) error {
		t.done = true

		return nil
	})

	return nil
}

func (t *Task) Cleanup(ctx context.Context) error {
	t.client.Node().Stop(ctx)

	t.client = nil

	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	return t.done, nil
}
