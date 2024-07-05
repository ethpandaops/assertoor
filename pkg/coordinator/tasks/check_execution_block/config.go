package checkexecutionblock

type Config struct {
	headBlock string `yaml:"headBlock" json:"headBlock"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
