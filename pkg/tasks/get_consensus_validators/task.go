package getconsensusvalidators

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "get_consensus_validators"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Retrieves validators from the consensus layer matching specified criteria.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "validators",
				Type:        "array",
				Description: "Array of validator info objects (when outputFormat is 'full').",
			},
			{
				Name:        "pubkeys",
				Type:        "array",
				Description: "Array of validator public keys (when outputFormat is 'pubkeys').",
			},
			{
				Name:        "indices",
				Type:        "array",
				Description: "Array of validator indices (when outputFormat is 'indices').",
			},
			{
				Name:        "count",
				Type:        "int",
				Description: "Number of matching validators found.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

type ValidatorInfo struct {
	Index                 uint64 `json:"index"`
	Pubkey                string `json:"pubkey"`
	Balance               uint64 `json:"balance"`
	Status                string `json:"status"`
	EffectiveBalance      uint64 `json:"effectiveBalance"`
	WithdrawalCredentials string `json:"withdrawalCredentials"`
	ActivationEpoch       uint64 `json:"activationEpoch"`
	ExitEpoch             uint64 `json:"exitEpoch"`
	WithdrawableEpoch     uint64 `json:"withdrawableEpoch"`
	Slashed               bool   `json:"slashed"`
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

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

//nolint:gocyclo // ignore
func (t *Task) Execute(_ context.Context) error {
	// Get client pool and validator names
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	consensusPool := clientPool.GetConsensusPool()
	validatorNames := t.ctx.Scheduler.GetServices().ValidatorNames()

	// Get validator set from consensus client
	validatorSet := consensusPool.GetValidatorSet()
	if validatorSet == nil {
		return fmt.Errorf("validator set not available")
	}

	matchingValidators := []ValidatorInfo{}

	// Compile validator name pattern regex if provided
	var validatorNameRegex *regexp.Regexp

	if t.config.ValidatorNamePattern != "" {
		var err error

		validatorNameRegex, err = regexp.Compile(t.config.ValidatorNamePattern)
		if err != nil {
			return fmt.Errorf("invalid validator name pattern: %v", err)
		}
	}

	// Compile client pattern regex if provided
	var clientNameRegex *regexp.Regexp

	if t.config.ClientPattern != "" {
		var err error

		clientNameRegex, err = regexp.Compile(t.config.ClientPattern)
		if err != nil {
			return fmt.Errorf("invalid client pattern: %v", err)
		}
	}

	// Iterate through validators and apply filters
	for validatorIndex, validator := range validatorSet {
		if len(matchingValidators) >= t.config.MaxResults {
			break
		}

		// Check index range
		if t.config.MinValidatorIndex != nil && uint64(validatorIndex) < *t.config.MinValidatorIndex {
			continue
		}

		if t.config.MaxValidatorIndex != nil && uint64(validatorIndex) > *t.config.MaxValidatorIndex {
			continue
		}

		// Check balance range
		if t.config.MinValidatorBalance != nil && uint64(validator.Balance) < *t.config.MinValidatorBalance {
			continue
		}

		if t.config.MaxValidatorBalance != nil && uint64(validator.Balance) > *t.config.MaxValidatorBalance {
			continue
		}

		// Check withdrawal credentials prefix
		if t.config.WithdrawalCredsPrefix != "" {
			credsHex := fmt.Sprintf("0x%x", validator.Validator.WithdrawalCredentials)
			if !strings.HasPrefix(credsHex, t.config.WithdrawalCredsPrefix) {
				continue
			}
		}

		// Check validator status
		if len(t.config.ValidatorStatus) > 0 {
			statusMatch := false
			validatorStatus := validator.Status.String()

			for _, allowedStatus := range t.config.ValidatorStatus {
				if validatorStatus == allowedStatus {
					statusMatch = true
					break
				}
			}

			if !statusMatch {
				continue
			}
		}

		// Get validator name for pattern matching
		validatorName := ""
		if validatorNames != nil {
			validatorName = validatorNames.GetValidatorName(uint64(validatorIndex))
		}

		// Check validator name pattern
		if validatorNameRegex != nil && validatorName != "" {
			if !validatorNameRegex.MatchString(validatorName) {
				continue
			}
		}

		// Check client pattern (match validator name against client pattern)
		if clientNameRegex != nil && validatorName != "" {
			if !clientNameRegex.MatchString(validatorName) {
				continue
			}
		}

		// Create validator info
		validatorInfo := ValidatorInfo{
			Index:                 uint64(validatorIndex),
			Pubkey:                fmt.Sprintf("0x%x", validator.Validator.PublicKey),
			Balance:               uint64(validator.Balance),
			Status:                validator.Status.String(),
			EffectiveBalance:      uint64(validator.Validator.EffectiveBalance),
			WithdrawalCredentials: fmt.Sprintf("0x%x", validator.Validator.WithdrawalCredentials),
			ActivationEpoch:       uint64(validator.Validator.ActivationEpoch),
			ExitEpoch:             uint64(validator.Validator.ExitEpoch),
			WithdrawableEpoch:     uint64(validator.Validator.WithdrawableEpoch),
			Slashed:               validator.Validator.Slashed,
		}

		matchingValidators = append(matchingValidators, validatorInfo)
	}

	// Set outputs based on format
	switch t.config.OutputFormat {
	case "full":
		if validatorsData, err := vars.GeneralizeData(matchingValidators); err == nil {
			t.ctx.Outputs.SetVar("validators", validatorsData)
		} else {
			return fmt.Errorf("failed to generalize validators data: %v", err)
		}
	case "pubkeys":
		pubkeys := make([]string, len(matchingValidators))
		for i, validator := range matchingValidators {
			pubkeys[i] = validator.Pubkey
		}

		if pubkeysData, err := vars.GeneralizeData(pubkeys); err == nil {
			t.ctx.Outputs.SetVar("pubkeys", pubkeysData)
		} else {
			return fmt.Errorf("failed to generalize pubkeys data: %v", err)
		}
	case "indices":
		indices := make([]uint64, len(matchingValidators))
		for i, validator := range matchingValidators {
			indices[i] = validator.Index
		}

		if indicesData, err := vars.GeneralizeData(indices); err == nil {
			t.ctx.Outputs.SetVar("indices", indicesData)
		} else {
			return fmt.Errorf("failed to generalize indices data: %v", err)
		}
	}

	// Always set count
	t.ctx.Outputs.SetVar("count", len(matchingValidators))

	t.logger.Infof("Found %d validators matching criteria", len(matchingValidators))

	if len(matchingValidators) > 0 {
		t.ctx.SetResult(types.TaskResultSuccess)
	} else {
		t.ctx.SetResult(types.TaskResultNone)
	}

	t.ctx.ReportProgress(100, fmt.Sprintf("Retrieved %d validators", len(matchingValidators)))

	return nil
}
