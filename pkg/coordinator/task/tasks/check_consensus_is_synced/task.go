package checkconsensusissynced

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/minccino/pkg/coordinator/task/types"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_is_synced"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks consensus clients for their sync status.",
		Config:      Config{},
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	result  types.TaskResult
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	config := DefaultConfig()
	if options.Config != nil {
		conf := &Config{}
		if err := options.Config.Unmarshal(&conf); err != nil {
			return nil, fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
		if err := mergo.Merge(&config, conf, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging task config for %v: %w", TaskName, err)
		}
	}
	return &Task{
		ctx:     ctx,
		options: options,
		config:  config,
		logger:  ctx.Logger.WithField("task", TaskName),
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.options.Title
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) PollingInterval() time.Duration {
	return time.Second * 5
}

func (t *Task) GetResult() (types.TaskResult, error) {
	taskStatus := t.ctx.Scheduler.GetTaskStatus(t)
	return t.result, taskStatus.Error
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	/*
		t.client = consensus.GetConsensusClient(ctx, t.log, t.consensusURL)

		return t.client.Start(ctx)
	*/
	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *Task) Cleanup(ctx context.Context) error {

	return nil
}

func (t *Task) checkClient(ctx context.Context) (bool, error) {
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
