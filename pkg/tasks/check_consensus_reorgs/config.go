package checkconsensusreorgs

type Config struct {
	MinCheckEpochCount uint64  `yaml:"minCheckEpochCount" json:"minCheckEpochCount"`
	MaxReorgDistance   uint64  `yaml:"maxReorgDistance" json:"maxReorgDistance"`
	MaxReorgsPerEpoch  float64 `yaml:"maxReorgsPerEpoch" json:"maxReorgsPerEpoch"`
	MaxTotalReorgs     uint64  `yaml:"maxTotalReorgs" json:"maxTotalReorgs"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if reorgs occur.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}

func DefaultConfig() Config {
	return Config{
		MinCheckEpochCount: 1,
	}
}

func (c *Config) Validate() error {
	return nil
}
