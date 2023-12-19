package checkconsensusproposerduty

type Config struct {
	ValidatorNamePattern string  `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	ValidatorIndex       *uint64 `yaml:"validatorIndex" json:"validatorIndex"`
	MaxSlotDistance      uint64  `yaml:"maxSlotDistance" json:"maxSlotDistance"`
	FailOnCheckMiss      bool    `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
