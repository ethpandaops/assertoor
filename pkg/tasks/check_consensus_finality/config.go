package checkconsensusfinality

type Config struct {
	MinUnfinalizedEpochs uint64 `yaml:"minUnfinalizedEpochs" json:"minUnfinalizedEpochs" desc:"Minimum number of unfinalized epochs required to pass the check."`
	MaxUnfinalizedEpochs uint64 `yaml:"maxUnfinalizedEpochs" json:"maxUnfinalizedEpochs" desc:"Maximum number of unfinalized epochs allowed to pass the check."`
	MinFinalizedEpochs   uint64 `yaml:"minFinalizedEpochs" json:"minFinalizedEpochs" desc:"Minimum number of finalized epochs required to pass the check."`
	FailOnCheckMiss      bool   `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when the finality check condition is not met."`
	ContinueOnPass       bool   `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
