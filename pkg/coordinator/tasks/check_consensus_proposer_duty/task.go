package checkconsensusproposerduty

import (
	"context"
	"fmt"
	"regexp"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_proposer_duty"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check consensus chain proposer duties.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx                   *types.TaskContext
	options               *types.TaskOptions
	config                Config
	logger                logrus.FieldLogger
	currentProposerDuties map[uint64]*v1.ProposerDuty
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
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
	consensusPool := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool()

	wallclockEpochSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer wallclockEpochSubscription.Unsubscribe()

	wallclockSlotSubscription := consensusPool.GetBlockCache().SubscribeWallclockSlotEvent(10)
	defer wallclockSlotSubscription.Unsubscribe()

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return fmt.Errorf("failed fetching wallclock: %w", err)
	}

	// load current epoch duties
	t.loadEpochDuties(ctx, currentEpoch.Number())

	for {
		select {
		case currentEpoch := <-wallclockEpochSubscription.Channel():
			t.loadEpochDuties(ctx, currentEpoch.Number())

		case currentSlot := <-wallclockSlotSubscription.Channel():
			checkResult := t.runProposerDutyCheck(currentSlot.Number())

			switch {
			case checkResult:
				t.ctx.SetResult(types.TaskResultSuccess)
			case t.config.FailOnCheckMiss:
				t.ctx.SetResult(types.TaskResultFailure)
			default:
				t.ctx.SetResult(types.TaskResultNone)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) loadEpochDuties(ctx context.Context, epoch uint64) {
	client := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)
	proposerDuties, err := client.GetRPCClient().GetProposerDuties(ctx, epoch)

	if err != nil {
		t.logger.Errorf("error while fetching epoch duties: %v", err.Error())
		return
	}

	t.currentProposerDuties = make(map[uint64]*v1.ProposerDuty)
	for _, duty := range proposerDuties {
		t.currentProposerDuties[uint64(duty.Slot)] = duty
	}
}

func (t *Task) runProposerDutyCheck(slot uint64) bool {
	if t.currentProposerDuties == nil {
		t.logger.Errorf("slot %v check failed: no proposer duties", slot)
		return false
	}

	currentSlot := slot

	for {
		duty := t.currentProposerDuties[slot]
		if duty == nil {
			t.logger.Errorf("slot %v check failed: no matching duty", slot)
			return false
		}

		if t.config.MaxSlotDistance > 0 && slot-currentSlot > t.config.MaxSlotDistance {
			t.logger.Errorf("slot %v check failed: no matching duty in next %v slots", slot, t.config.MaxSlotDistance)
			return false
		}

		slot++

		if t.config.ValidatorIndex != nil && uint64(duty.ValidatorIndex) != *t.config.ValidatorIndex {
			continue
		}

		if t.config.ValidatorNamePattern != "" {
			validatorName := t.ctx.Scheduler.GetCoordinator().ValidatorNames().GetValidatorName(uint64(duty.ValidatorIndex))
			matched, err := regexp.MatchString(t.config.ValidatorNamePattern, validatorName)

			if err != nil {
				t.logger.Errorf("slot %v check failed: validator name pattern invalid: %v", slot, err)
				return false
			}

			if !matched {
				continue
			}
		}

		if t.config.MinSlotDistance > 0 && slot-1-currentSlot < t.config.MinSlotDistance {
			t.logger.Errorf("slot %v check failed: matching duty too early: in %v slots, min distance: %v", slot, slot-1-currentSlot, t.config.MinSlotDistance)
			return false
		}

		return true
	}
}
