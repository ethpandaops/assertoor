package checkconsensusreorgs

type Config struct {
	MinCheckEpochCount uint64  `yaml:"minCheckEpochCount" json:"minCheckEpochCount"`
	MaxReorgDistance   uint64  `yaml:"maxReorgDistance" json:"maxReorgDistance"`
	MaxReorgsPerEpoch  float64 `yaml:"maxReorgsPerEpoch" json:"maxReorgsPerEpoch"`
	MaxTotalReorgs     uint64  `yaml:"maxTotalReorgs" json:"maxTotalReorgs"`
}

func DefaultConfig() Config {
	return Config{
		MinCheckEpochCount: 1,
	}
}

func (c *Config) Validate() error {
	return nil
}
