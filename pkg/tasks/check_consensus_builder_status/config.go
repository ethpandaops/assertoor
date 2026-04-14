package checkconsensusbuildersstatus

type Config struct {
	BuilderPubKey     string  `yaml:"builderPubKey" json:"builderPubKey" desc:"Public key of the builder to check."`
	BuilderIndex      *uint64 `yaml:"builderIndex" json:"builderIndex" desc:"Index of the builder to check."`
	MinBuilderBalance uint64  `yaml:"minBuilderBalance" json:"minBuilderBalance" desc:"Minimum builder balance required (in gwei)."`
	MaxBuilderBalance *uint64 `yaml:"maxBuilderBalance" json:"maxBuilderBalance" desc:"Maximum builder balance allowed (in gwei)."`
	ExpectExiting     bool    `yaml:"expectExiting" json:"expectExiting" desc:"If true, expect the builder to have a non-FAR_FUTURE withdrawable epoch (i.e. exiting or exited)."`
	ExpectActive      bool    `yaml:"expectActive" json:"expectActive" desc:"If true, expect the builder to have FAR_FUTURE withdrawable epoch (i.e. active)."`
	FailOnCheckMiss   bool    `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when builder status check condition is not met."`
	ContinueOnPass    bool    `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
	ClientPattern     string  `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for state queries."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
