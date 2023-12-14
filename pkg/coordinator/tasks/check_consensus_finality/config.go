package checkconsensusfinality

type Config struct {
	MinUnfinalizedEpochs uint64 `yaml:"minUnfinalizedEpochs" json:"minUnfinalizedEpochs"`
	MaxUnfinalizedEpochs uint64 `yaml:"maxUnfinalizedEpochs" json:"maxUnfinalizedEpochs"`
	MinFinalizedEpochs   uint64 `yaml:"minFinalizedEpochs" json:"minFinalizedEpochs"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
