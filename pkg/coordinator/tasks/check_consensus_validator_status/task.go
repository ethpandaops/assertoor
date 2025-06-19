package checkconsensusvalidatorstatus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	"github.com/ethereum/go-ethereum/common"
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

func (t *Task) Config() interface{} {
	return t.config
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
	matchingValidators := uint64(0)
	pubkey := []byte{}

	var namePattern *regexp.Regexp

	if t.config.ValidatorPubKey != "" {
		pubkey = common.FromHex(t.config.ValidatorPubKey)
	}

	if t.config.ValidatorNamePattern != "" {
		pattern, err := regexp.Compile(t.config.ValidatorNamePattern)
		if err != nil {
			t.logger.Errorf("check failed: validator name pattern invalid: %v", err)
			return false
		}

		namePattern = pattern
	}

	for {
		validator := validatorSet[phase0.ValidatorIndex(currentIndex)]
		if validator == nil {
			break
		}

		currentIndex++

		if t.config.ValidatorIndex != nil && uint64(validator.Index) != *t.config.ValidatorIndex {
			continue
		}

		if t.config.ValidatorNamePattern != "" && !namePattern.MatchString(t.ctx.Scheduler.GetServices().ValidatorNames().GetValidatorName(uint64(validator.Index))) {
			continue
		}

		if t.config.ValidatorPubKey != "" && !bytes.Equal(pubkey, validator.Validator.PublicKey[:]) {
			continue
		}

		// found a matching validator
		t.logger.Infof("validator found: index %v, status: %v", validator.Index, validator.Status.String())

		matchingValidators++

		if body, err := vars.GeneralizeData(validator); err == nil {
			t.ctx.Outputs.SetVar("validator", body)
		} else {
			t.logger.Warnf("Failed encoding validator info for validator output: %v", err)
		}

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

		t.ctx.Outputs.SetVar("pubkey", fmt.Sprintf("0x%x", validator.Validator.PublicKey[:]))

		if t.config.ValidatorPubKeyResultVar != "" {
			t.ctx.Vars.SetVar(t.config.ValidatorPubKeyResultVar, fmt.Sprintf("0x%x", validator.Validator.PublicKey[:]))
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

		if t.config.MinValidatorBalance > 0 && validator.Balance < phase0.Gwei(t.config.MinValidatorBalance) {
			t.logger.Infof("check failed: validator balance below minimum: %v", validator.Balance)
			continue
		}

		if t.config.MaxValidatorBalance != nil && validator.Balance > phase0.Gwei(*t.config.MaxValidatorBalance) {
			t.logger.Infof("check failed: validator balance above maximum: %v", validator.Balance)
			continue
		}

		if t.config.WithdrawalCredsPrefix != "" && !bytes.HasPrefix(validator.Validator.WithdrawalCredentials, common.FromHex(t.config.WithdrawalCredsPrefix)) {
			t.logger.Infof("check failed: withdrawal creds prefix mismatch")
			continue
		}

		return true
	}

	if matchingValidators == 0 {
		t.logger.Infof("check failed: no matching validator found")
	}

	return false
}
