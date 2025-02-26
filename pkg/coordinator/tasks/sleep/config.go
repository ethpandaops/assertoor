package sleep

import (
	"errors"

	"github.com/noku-team/assertoor/pkg/coordinator/helper"
)

type Config struct {
	Duration helper.Duration `yaml:"duration" json:"duration"`
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
