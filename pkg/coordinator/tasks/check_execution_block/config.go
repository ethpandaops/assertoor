package checkexecutionblock

type Config struct {
	BlockHeaderResultVar string `yaml:"BlockHeaderResultVar" json:"BlockHeaderResultVar"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
