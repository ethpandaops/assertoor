package runtasks

import (
	"errors"

	"github.com/ethpandaops/minccino/pkg/coordinator/helper"
)

type Config struct {
	// child tasks
	Tasks []helper.RawMessage `yaml:"tasks" json:"tasks"`
}

func DefaultConfig() Config {
	return Config{
		Tasks: []helper.RawMessage{},
	}
}

func (c *Config) Validate() error {
	if len(c.Tasks) == 0 {
		return errors.New("at least one task must be specified")
	}
	return nil
}
