package sleep

import "time"

type Config struct {
	Duration time.Duration `yaml:"duration" json:"duration"`
}

func DefaultConfig() Config {
	return Config{
		Duration: time.Second * 1,
	}
}

func (c *Config) Validate() error {
	return nil
}
