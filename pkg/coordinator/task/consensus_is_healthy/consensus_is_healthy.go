package consensusishealthy

import (
	"context"
	"time"

	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type Task struct {
	log          logrus.FieldLogger
	client       *consensus.Client
	consensusURL string
	config       Config
	title        string
	timeout      time.Duration
}

const (
	Name        = "consensus_is_healthy"
	Description = "Performs a health check against the consensus client."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, consensusURL string, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		consensusURL: consensusURL,
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

func (t *Task) PollingInterval() time.Duration {
	return time.Second * 1
}

func (t *Task) Start(ctx context.Context) error {
	t.client = consensus.GetConsensusClient(ctx, t.log, t.consensusURL)

	return nil
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

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
