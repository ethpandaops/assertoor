package checkconsensusvalidatorstatus

type Config struct {
	ValidatorPubKey       string   `yaml:"validatorPubKey" json:"validatorPubKey"`
	ValidatorNamePattern  string   `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	ValidatorIndex        *uint64  `yaml:"validatorIndex" json:"validatorIndex"`
	ValidatorStatus       []string `yaml:"validatorStatus" json:"validatorStatus"`
	MinValidatorBalance   uint64   `yaml:"minValidatorBalance" json:"minValidatorBalance"`
	MaxValidatorBalance   *uint64  `yaml:"maxValidatorBalance" json:"maxValidatorBalance"`
	WithdrawalCredsPrefix string   `yaml:"withdrawalCredsPrefix" json:"withdrawalCredsPrefix"`
	FailOnCheckMiss       bool     `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`

	ValidatorInfoResultVar   string `yaml:"validatorInfoResultVar" json:"validatorInfoResultVar"`
	ValidatorPubKeyResultVar string `yaml:"validatorPubKeyResultVar" json:"validatorPubKeyResultVar"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
