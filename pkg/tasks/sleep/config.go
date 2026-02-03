package sleep

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Duration helper.Duration `yaml:"duration" json:"duration" require:"A" desc:"Duration to sleep (e.g., '10s', '5m', '1h')."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.Duration.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}

	return nil
}
