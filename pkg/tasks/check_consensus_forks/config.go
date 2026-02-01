package checkconsensusforks

type Config struct {
	MinCheckEpochCount uint64 `yaml:"minCheckEpochCount" json:"minCheckEpochCount"`
	MaxForkDistance    int64  `yaml:"maxForkDistance" json:"maxForkDistance"`
	MaxForkCount       uint64 `yaml:"maxForkCount" json:"maxForkCount"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if forks occur.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}

func DefaultConfig() Config {
	return Config{
		MinCheckEpochCount: 1,
		MaxForkDistance:    1,
	}
}

func (c *Config) Validate() error {
	return nil
}
