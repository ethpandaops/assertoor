package checkconsensusproposerduty

type Config struct {
	ValidatorNamePattern string  `yaml:"validatorNamePattern" json:"validatorNamePattern" desc:"Regex pattern to match validator names for proposer duty check."`
	ValidatorIndex       *uint64 `yaml:"validatorIndex" json:"validatorIndex" desc:"Specific validator index to check for proposer duty."`
	MinSlotDistance      uint64  `yaml:"minSlotDistance" json:"minSlotDistance" desc:"Minimum slot distance from current slot for proposer duty."`
	MaxSlotDistance      uint64  `yaml:"maxSlotDistance" json:"maxSlotDistance" desc:"Maximum slot distance from current slot for proposer duty."`
	FailOnCheckMiss      bool    `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when no proposer duty is found in range."`
	ContinueOnPass       bool    `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
