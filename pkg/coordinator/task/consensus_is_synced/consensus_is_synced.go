package consensusissynced

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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
	Name        = "consensus_is_synced"
	Description = "Waits until the consensus client considers itself synced."
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

func (t *Task) Title() string {
	return t.title
}

func (t *Task) Timeout() time.Duration {
	return t.timeout
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

	if status.Percent() < t.config.Percent {
		return false, nil
	}

	if !t.config.WaitForChainProgression {
		return true, nil
	}

	t.log.WithField("min_slot_height", t.config.MinSlotHeight).Info("Waiting for head slot to advance")

	block, err := t.client.Node().FetchBlock(ctx, "head")
	if err != nil {
		return false, err
	}

	slot, err := block.Slot()
	if err != nil {
		return false, err
	}

	if slot > phase0.Slot(t.config.MinSlotHeight) {
		return true, nil
	}

	t.log.WithField("slot", slot).Info("Head slot has not advanced far enough")

	return false, nil
}
