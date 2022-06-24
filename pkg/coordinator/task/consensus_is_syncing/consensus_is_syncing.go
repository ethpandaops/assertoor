package consensusissyncing

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type Task struct {
	consensusURL string
	client       *consensus.Client
	log          logrus.FieldLogger
	config       Config
	title        string
	timeout      time.Duration
}

const (
	Name        = "consensus_is_syncing"
	Description = "Waits until the consensus client considers itself syncing."
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
	return time.Second * 5
}

func (t *Task) Start(ctx context.Context) error {
	t.client = consensus.GetConsensusClient(ctx, t.log, t.consensusURL)

	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	status, err := t.client.GetSyncStatus(ctx)
	if err != nil {
		return false, err
	}

	return status.IsSyncing, err
}
