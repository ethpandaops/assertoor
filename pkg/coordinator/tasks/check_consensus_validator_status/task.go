package checkconsensusvalidatorstatus

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_validator_status"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check validator status on consensus chain.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx                 *types.TaskContext
	options             *types.TaskOptions
	config              Config
	logger              logrus.FieldLogger
	currentValidatorSet map[uint64]*v1.Validator
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

	// load current epoch duties
	t.loadValidatorSet(ctx)

	for {
		select {
		case <-wallclockEpochSubscription.Channel():
			t.loadValidatorSet(ctx)

			checkResult := t.runValidatorStatusCheck()

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

func (t *Task) loadValidatorSet(ctx context.Context) {
	client := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)
	validatorSet, err := client.GetRPCClient().GetStateValidators(ctx, "head")

	if err != nil {
		t.logger.Errorf("error while fetching validator set: %v", err.Error())
		return
	}

	t.currentValidatorSet = make(map[uint64]*v1.Validator)
	for _, val := range validatorSet {
		t.currentValidatorSet[uint64(val.Index)] = val
	}
}

func (t *Task) runValidatorStatusCheck() bool {
	if t.currentValidatorSet == nil {
		t.logger.Errorf("check failed: no validator set")
		return false
	}

	currentIndex := uint64(0)

	for {
		validator := t.currentValidatorSet[currentIndex]
		if validator == nil {
			t.logger.Errorf("check failed: no matching validator found")
			return false
		}

		currentIndex++

		if t.config.ValidatorIndex != nil && uint64(validator.Index) != *t.config.ValidatorIndex {
			continue
		}

		if t.config.ValidatorNamePattern != "" {
			validatorName := t.ctx.Scheduler.GetCoordinator().ValidatorNames().GetValidatorName(uint64(validator.Index))
			matched, err := regexp.MatchString(t.config.ValidatorNamePattern, validatorName)

			if err != nil {
				t.logger.Errorf("check failed: validator name pattern invalid: %v", err)
				return false
			}

			if !matched {
				continue
			}
		}

		if t.config.ValidatorPubKey != "" {
			pubkey := common.FromHex(t.config.ValidatorPubKey)

			if !bytes.Equal(pubkey, validator.Validator.PublicKey[:]) {
				continue
			}
		}

		if len(t.config.ValidatorStatus) > 0 {
			found := false

			for _, s := range t.config.ValidatorStatus {
				if validator.Status.String() == s {
					found = true
					break
				}
			}

			if !found {
				continue
			}
		}

		return true
	}
}
