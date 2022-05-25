package executionhasprogressed

import "errors"

type Config struct {
	Distance int64 `yaml:"distance" json:"distance"`
}

func DefaultConfig() Config {
	return Config{
		Distance: 3,
	}
}

func (c *Config) Validate() error {
	if c.Distance < 0 {
		return errors.New("distance must be >= 0")
	}

	return nil
}
