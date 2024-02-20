package checkconsensusvalidatorstatus

type Config struct {
	ValidatorPubKey      string   `yaml:"validatorPubKey" json:"validatorPubKey"`
	ValidatorNamePattern string   `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	ValidatorIndex       *uint64  `yaml:"validatorIndex" json:"validatorIndex"`
	ValidatorStatus      []string `yaml:"validatorStatus" json:"validatorStatus"`
	FailOnCheckMiss      bool     `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`

	ValidatorInfoResultVar string `yaml:"validatorInfoResultVar" json:"validatorInfoResultVar"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
