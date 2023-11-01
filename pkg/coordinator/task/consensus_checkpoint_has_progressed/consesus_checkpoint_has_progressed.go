package consensuscheckpointhasprogressed

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/sirupsen/logrus"
)

type Task struct {
	client       *consensus.Client
	log          logrus.FieldLogger
	config       Config
	title        string
	timeout      time.Duration
	consensusURL string

	checkpoint *phase0.Epoch
}

const (
	Name        = "consensus_checkpoint_has_progressed"
	Description = "Checks if a consensus checkpoint has progressed (i.e. if the 'finalized' checkpoint has advanced by 2 epochs.)."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, consensusURL string, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		consensusURL: consensusURL,
		log:          log.WithField("task", Name).WithField("checkpoint_name", config.CheckpointName),
		config:       config,
		title:        title,
		timeout:      timeout,

		checkpoint: nil,
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

	return t.client.Start(ctx)
}

func (t *Task) Cleanup(ctx context.Context) error {
	t.client.Node().Stop(ctx)

	t.client = nil

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

	// If the checkpoint isn't set then this is our starting epich.
	if t.checkpoint == nil {
		t.checkpoint = &checkpoint.Epoch

		return false, nil
	}

	// If the checkpoint hasn't changed, we're still waiting.
	if t.checkpoint == &checkpoint.Epoch {
		return false, nil
	}

	if checkpoint.Epoch-*t.checkpoint >= t.config.Distance {
		return true, nil
	}

	return true, nil
}
