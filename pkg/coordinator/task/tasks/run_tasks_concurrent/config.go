package runtasksconcurrent

import (
	"errors"

	"github.com/ethpandaops/minccino/pkg/coordinator/helper"
)

type Config struct {
	// number of successfull child tasks to make this task succeed (0 = all tasks)
	SucceedTaskCount uint64 `yaml:"succeedTaskCount" json:"succeedTaskCount"`

	// number of failed child tasks to make this task fail (0 = all tasks)
	FailTaskCount uint64 `yaml:"failTaskCount" json:"failTaskCount"`

	// child tasks
	Tasks []helper.RawMessage `yaml:"tasks" json:"tasks"`
}

func DefaultConfig() Config {
	return Config{
		Tasks:         []helper.RawMessage{},
		FailTaskCount: 1,
	}
}

func (c *Config) Validate() error {
	if len(c.Tasks) == 0 {
		return errors.New("at least one task must be specified")
	}
	return nil
}
