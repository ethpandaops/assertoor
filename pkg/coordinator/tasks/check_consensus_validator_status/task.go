package checkconsensusvalidatorstatus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
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
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
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
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	wallclockEpochSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer wallclockEpochSubscription.Unsubscribe()

	// load current epoch duties
	t.runCheck()

	for {
		select {
		case <-wallclockEpochSubscription.Channel():
			t.runCheck()
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) runCheck() {
	checkResult := t.runValidatorStatusCheck()

	_, epoch, _ := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().GetWallclock().Now()
	t.logger.Infof("epoch %v check result: %v", epoch.Number(), checkResult)

	switch {
	case checkResult:
		t.ctx.SetResult(types.TaskResultSuccess)
	case t.config.FailOnCheckMiss:
		t.ctx.SetResult(types.TaskResultFailure)
	default:
		t.ctx.SetResult(types.TaskResultNone)
	}
}

func (t *Task) runValidatorStatusCheck() bool {
	validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()
	if validatorSet == nil {
		t.logger.Errorf("check failed: no validator set")
		return false
	}

	currentIndex := uint64(0)

	for {
		validator := validatorSet[phase0.ValidatorIndex(currentIndex)]
		if validator == nil {
			return false
		}

		currentIndex++

		if t.config.ValidatorIndex != nil && uint64(validator.Index) != *t.config.ValidatorIndex {
			continue
		}

		if t.config.ValidatorNamePattern != "" {
			validatorName := t.ctx.Scheduler.GetServices().ValidatorNames().GetValidatorName(uint64(validator.Index))
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
				t.logger.Infof("check failed: no matching validator found")
				continue
			}
		}

		// found a matching validator
		t.logger.Infof("validator found: index %v, status: %v", validator.Index, validator.Status.String())

		if t.config.ValidatorInfoResultVar != "" {
			validatorJSON, err := json.Marshal(validator)
			if err == nil {
				validatorMap := map[string]interface{}{}
				err = json.Unmarshal(validatorJSON, &validatorMap)

				if err == nil {
					t.ctx.Vars.SetVar(t.config.ValidatorInfoResultVar, validatorMap)
				} else {
					t.logger.Errorf("could not unmarshal validator info for result var: %v", err)
				}
			} else {
				t.logger.Errorf("could not marshal validator info for result var: %v", err)
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
