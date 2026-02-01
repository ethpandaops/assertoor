package checkconsensusfinality

type Config struct {
	MinUnfinalizedEpochs uint64 `yaml:"minUnfinalizedEpochs" json:"minUnfinalizedEpochs"`
	MaxUnfinalizedEpochs uint64 `yaml:"maxUnfinalizedEpochs" json:"maxUnfinalizedEpochs"`
	MinFinalizedEpochs   uint64 `yaml:"minFinalizedEpochs" json:"minFinalizedEpochs"`
	FailOnCheckMiss      bool   `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`
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
