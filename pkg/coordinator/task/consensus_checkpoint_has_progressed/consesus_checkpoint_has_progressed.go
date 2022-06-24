package consensuscheckpointhasprogressed

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type Task struct {
	client       *consensus.Client
	log          logrus.FieldLogger
	config       Config
	title        string
	timeout      time.Duration
	consensusURL string

	checkpoint int64
}

const (
	Name        = "consensus_checkpoint_has_progressed"
	Description = "Checks if a consensus checkpoint has progressed (i.e. if the `head` slot has advanced by 3)."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, consensusURL string, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		consensusURL: consensusURL,
		log:          log.WithField("task", Name).WithField("checkpoint_name", config.CheckpointName),
		config:       config,
		title:        title,
		timeout:      timeout,

		checkpoint: -1,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.timeout
}

func (t *Task) Title() string {
	return t.title
}

func (t *Task) Description() string {
	return Description
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

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	if _, err := t.client.GetSpec(ctx); err != nil {
		return false, err
	}

	checkpoint, err := t.client.GetCheckpoint(ctx, t.config.CheckpointName)
	if err != nil {
		return false, err
	}

	t.log.WithFields(logrus.Fields{
		"checkpoint":          checkpoint,
		"internal_checkpoint": t.checkpoint,
	}).Info("checking if checkpoint has progressed")

	// If the checkpoint is -1, we haven't gone through a cycle yet.
	if t.checkpoint == -1 {
		t.checkpoint = int64(checkpoint.Slot)

		return false, nil
	}

	// If the checkpoint hasn't changed, we're still waiting.
	if t.checkpoint == int64(checkpoint.Slot) {
		return false, nil
	}

	if int64(checkpoint.Slot)-t.checkpoint >= t.config.Distance {
		return true, nil
	}

	return true, nil
}
