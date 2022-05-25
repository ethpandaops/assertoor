package botharesynced

import (
	"context"
	"time"

	consensusissynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_synced"
	executionissynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_is_synced"
	"github.com/sirupsen/logrus"
)

type Task struct {
	log       logrus.FieldLogger
	execution *executionissynced.Task
	consensus *consensusissynced.Task
	config    Config
}

const (
	Name        = "both_are_synced"
	Description = "Waits until both consensus and execution clients are considered synced."
)

// NewTask returns a new BothAreSynced task.
func NewTask(ctx context.Context, log logrus.FieldLogger, consensusURL, executionURL string, config Config) *Task {
	consensus := consensusissynced.NewTask(ctx, log, consensusURL, config.ConsensusIsSyncedConfig)
	execution := executionissynced.NewTask(ctx, log, executionURL, config.ExecutionIsSyncedConfig)

	return &Task{
		log:       log.WithField("task", Name),
		consensus: consensus,
		execution: execution,
		config:    config,
	}
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) PollingInterval() time.Duration {
	return time.Second * 5
}

func (t *Task) Start(ctx context.Context) error {
	if err := t.consensus.Start(ctx); err != nil {
		return err
	}

	if err := t.execution.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) ValidateConfig() error {
	if err := t.consensus.ValidateConfig(); err != nil {
		return err
	}

	if err := t.execution.ValidateConfig(); err != nil {
		return err
	}

	return nil
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	execution, err := t.execution.IsComplete(ctx)
	if err != nil {
		t.log.WithError(err).Error("failed to check execution client")
	}

	consensus, err := t.consensus.IsComplete(ctx)
	if err != nil {
		t.log.WithError(err).Error("failed to check consensus client")
	}

	t.log.WithFields(
		logrus.Fields{
			"execution": execution,
			"consensus": consensus,
		},
	).Info("checked both clients")

	if !consensus || !execution {
		return false, nil
	}

	return true, nil
}
