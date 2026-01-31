package getconsensusvalidators

import (
	"fmt"
)

type Config struct {
	ClientPattern         string   `yaml:"clientPattern" json:"clientPattern"`
	ValidatorNamePattern  string   `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	ValidatorStatus       []string `yaml:"validatorStatus" json:"validatorStatus"`
	MinValidatorBalance   *uint64  `yaml:"minValidatorBalance" json:"minValidatorBalance"`
	MaxValidatorBalance   *uint64  `yaml:"maxValidatorBalance" json:"maxValidatorBalance"`
	WithdrawalCredsPrefix string   `yaml:"withdrawalCredsPrefix" json:"withdrawalCredsPrefix"`
	MinValidatorIndex     *uint64  `yaml:"minValidatorIndex" json:"minValidatorIndex"`
	MaxValidatorIndex     *uint64  `yaml:"maxValidatorIndex" json:"maxValidatorIndex"`
	MaxResults            int      `yaml:"maxResults" json:"maxResults"`

	// Output format options
	OutputFormat string `yaml:"outputFormat" json:"outputFormat"` // "full", "pubkeys", "indices"
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
