package checkconsensusforks

type Config struct {
	MinCheckEpochCount uint64 `yaml:"minCheckEpochCount" json:"minCheckEpochCount" desc:"Minimum number of epochs to monitor before evaluating fork conditions."`
	MaxForkDistance    int64  `yaml:"maxForkDistance" json:"maxForkDistance" desc:"Maximum allowed fork distance (depth) in slots."`
	MaxForkCount       uint64 `yaml:"maxForkCount" json:"maxForkCount" desc:"Maximum number of forks allowed during monitoring."`
	ContinueOnPass     bool   `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
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
