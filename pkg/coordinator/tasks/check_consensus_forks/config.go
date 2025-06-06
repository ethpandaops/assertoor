package checkconsensusforks

type Config struct {
	MinCheckEpochCount uint64 `yaml:"minCheckEpochCount" json:"minCheckEpochCount"`
	MaxForkDistance    int64  `yaml:"maxForkDistance" json:"maxForkDistance"`
	MaxForkCount       uint64 `yaml:"maxForkCount" json:"maxForkCount"`
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
