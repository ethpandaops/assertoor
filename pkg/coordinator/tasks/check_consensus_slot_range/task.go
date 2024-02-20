package checkconsensusslotrange

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_slot_range"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check if consensus wallclock is in a specific range.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx         *types.TaskContext
	options     *types.TaskOptions
	config      Config
	logger      logrus.FieldLogger
	firstHeight map[uint16]uint64
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:         ctx,
		options:     options,
		logger:      ctx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	wallclockSubscription := consensusPool.GetBlockCache().SubscribeWallclockSlotEvent(10)
	defer wallclockSubscription.Unsubscribe()

	for {
		checkResult, isLower := t.runRangeCheck()

		switch {
		case checkResult:
			t.ctx.SetResult(types.TaskResultSuccess)
		case !isLower || t.config.FailIfLower:
			t.ctx.SetResult(types.TaskResultFailure)
		default:
			t.ctx.SetResult(types.TaskResultNone)
		}

		select {
		case slot := <-wallclockSubscription.Channel():
			t.logger.Infof("wallclock slot %v", slot.Number())
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) runRangeCheck() (checkResult, isLower bool) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	currentSlot, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		t.logger.Errorf("cannot fetch wallclock: %v", err)
		return false, true
	}

	if currentSlot.Number() < t.config.MinSlotNumber {
		return false, true
	}

	if currentEpoch.Number() < t.config.MinEpochNumber {
		return false, true
	}

	if currentSlot.Number() > t.config.MaxSlotNumber {
		return false, false
	}

	if currentEpoch.Number() > t.config.MaxEpochNumber {
		return false, false
	}

	return true, false
}
