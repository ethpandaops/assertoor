package checkconsensusvalidatorstatus

type Config struct {
	ValidatorPubKey       string   `yaml:"validatorPubKey" json:"validatorPubKey" desc:"Public key of the validator to check."`
	ValidatorNamePattern  string   `yaml:"validatorNamePattern" json:"validatorNamePattern" desc:"Regex pattern to match validator names."`
	ValidatorIndex        *uint64  `yaml:"validatorIndex" json:"validatorIndex" desc:"Index of the validator to check."`
	ValidatorStatus       []string `yaml:"validatorStatus" json:"validatorStatus" desc:"List of expected validator statuses."`
	MinValidatorBalance   uint64   `yaml:"minValidatorBalance" json:"minValidatorBalance" desc:"Minimum validator balance required (in gwei)."`
	MaxValidatorBalance   *uint64  `yaml:"maxValidatorBalance" json:"maxValidatorBalance" desc:"Maximum validator balance allowed (in gwei)."`
	WithdrawalCredsPrefix string   `yaml:"withdrawalCredsPrefix" json:"withdrawalCredsPrefix" desc:"Expected prefix of withdrawal credentials."`
	FailOnCheckMiss       bool     `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when validator status check condition is not met."`
	ContinueOnPass        bool     `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`

	ValidatorInfoResultVar   string `yaml:"validatorInfoResultVar" json:"validatorInfoResultVar" desc:"Variable name to store the validator info."`
	ValidatorPubKeyResultVar string `yaml:"validatorPubKeyResultVar" json:"validatorPubKeyResultVar" desc:"Variable name to store the validator public key."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
