package getconsensusvalidators

import (
	"fmt"
)

type Config struct {
	ClientPattern         string   `yaml:"clientPattern" json:"clientPattern" require:"A.1" desc:"Regex pattern to select specific client endpoints for querying validators."`
	ValidatorNamePattern  string   `yaml:"validatorNamePattern" json:"validatorNamePattern" require:"A.2" desc:"Regex pattern to filter validators by name."`
	ValidatorStatus       []string `yaml:"validatorStatus" json:"validatorStatus" desc:"List of validator statuses to include in results."`
	MinValidatorBalance   *uint64  `yaml:"minValidatorBalance" json:"minValidatorBalance" desc:"Minimum validator balance to include in results."`
	MaxValidatorBalance   *uint64  `yaml:"maxValidatorBalance" json:"maxValidatorBalance" desc:"Maximum validator balance to include in results."`
	WithdrawalCredsPrefix string   `yaml:"withdrawalCredsPrefix" json:"withdrawalCredsPrefix" desc:"Prefix of withdrawal credentials to filter validators."`
	MinValidatorIndex     *uint64  `yaml:"minValidatorIndex" json:"minValidatorIndex" desc:"Minimum validator index to include in results."`
	MaxValidatorIndex     *uint64  `yaml:"maxValidatorIndex" json:"maxValidatorIndex" desc:"Maximum validator index to include in results."`
	MaxResults            int      `yaml:"maxResults" json:"maxResults" desc:"Maximum number of validators to return."`

	// Output format options
	OutputFormat string `yaml:"outputFormat" json:"outputFormat" desc:"Output format: 'full' (complete validator data), 'pubkeys' (public keys only), or 'indices' (validator indices only)."`
}

func DefaultConfig() Config {
	return Config{
		MaxResults:   100,
		OutputFormat: "full",
		ValidatorStatus: []string{
			"pending_initialized",
			"pending_queued",
			"active_ongoing",
			"active_exiting",
			"active_slashed",
			"exited_unslashed",
			"exited_slashed",
			"withdrawal_possible",
			"withdrawal_done",
		},
	}
}

func (c *Config) Validate() error {
	if c.ClientPattern == "" && c.ValidatorNamePattern == "" {
		return fmt.Errorf("either clientPattern or validatorNamePattern is required")
	}

	if c.MaxResults <= 0 {
		return fmt.Errorf("maxResults must be > 0")
	}

	if c.MinValidatorIndex != nil && c.MaxValidatorIndex != nil && *c.MinValidatorIndex > *c.MaxValidatorIndex {
		return fmt.Errorf("minValidatorIndex must be <= maxValidatorIndex")
	}

	if c.MinValidatorBalance != nil && c.MaxValidatorBalance != nil && *c.MinValidatorBalance > *c.MaxValidatorBalance {
		return fmt.Errorf("minValidatorBalance must be <= maxValidatorBalance")
	}

	validFormats := map[string]bool{"full": true, "pubkeys": true, "indices": true}
	if !validFormats[c.OutputFormat] {
		return fmt.Errorf("outputFormat must be one of: full, pubkeys, indices")
	}

	return nil
}
