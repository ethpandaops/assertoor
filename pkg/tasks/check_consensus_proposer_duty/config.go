package checkconsensusproposerduty

type Config struct {
	ValidatorNamePattern string  `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	ValidatorIndex       *uint64 `yaml:"validatorIndex" json:"validatorIndex"`
	MinSlotDistance      uint64  `yaml:"minSlotDistance" json:"minSlotDistance"`
	MaxSlotDistance      uint64  `yaml:"maxSlotDistance" json:"maxSlotDistance"`
	FailOnCheckMiss      bool    `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if conditions change.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
