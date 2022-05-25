package consensusissynced

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
}

const (
	Name        = "consensus_is_synced"
	Description = "Waits until the consensus client considers itself synced."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, consensusURL string, config Config) *Task {
	return &Task{
		consensusURL: consensusURL,
		log:          log.WithField("task", Name),
		config:       config,
	}
}

func DefaultConfig() Config {
	return Config{
		Percent:                 100,
		WaitForChainProgression: true,
		MinSlotHeight:           10,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Description() string {
	return Description
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

	t.log.WithField("percent", status.Percent()).Info("Sync status")

	if status.Percent() >= t.config.Percent {
		return true, nil
	}

	if !t.config.WaitForChainProgression {
		return true, nil
	}

	// Check that our head slot is greater than the min slot height just to be sure.
	// Like if the node has just started up and hasn't started syncing yet.
	checkpoint, err := t.client.GetCheckpoint(ctx, consensus.Head)
	if err != nil {
		return false, err
	}

	return int(checkpoint.Slot) > t.config.MinSlotHeight, nil
}
