package sleep

import (
	"errors"

	human "github.com/samcm/sync-test-coordinator/pkg/coordinator/human-duration"
)

type Config struct {
	Duration human.Duration `yaml:"duration" json:"duration"`
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
