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
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if validator status changes.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`

	ValidatorInfoResultVar   string `yaml:"validatorInfoResultVar" json:"validatorInfoResultVar"`
	ValidatorPubKeyResultVar string `yaml:"validatorPubKeyResultVar" json:"validatorPubKeyResultVar"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
