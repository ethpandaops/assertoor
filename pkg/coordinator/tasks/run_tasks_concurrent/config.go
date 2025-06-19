package runtasksconcurrent

import (
	"errors"

	"github.com/erigontech/assertoor/pkg/coordinator/helper"
)

type Config struct {
	// number of successful child tasks to make this task succeed (0 = all tasks)
	SucceedTaskCount uint64 `yaml:"succeedTaskCount" json:"succeedTaskCount"`

	// number of failed child tasks to make this task fail (0 = all tasks)
	FailTaskCount uint64 `yaml:"failTaskCount" json:"failTaskCount"`

	// fail task if neither succeedTaskCount nor failTaskCount is reached, but all tasks completed
	FailOnUndecided bool `yaml:"failOnUndecided" json:"failOnUndecided"`

	// create a new variable scope for the child tasks
	NewVariableScope bool `yaml:"newVariableScope" json:"newVariableScope"`

	// child tasks
	Tasks []helper.RawMessageMasked `yaml:"tasks" json:"tasks"`
}

func DefaultConfig() Config {
	return Config{
		Tasks:            []helper.RawMessageMasked{},
		FailTaskCount:    1,
		NewVariableScope: true,
	}
}

func (c *Config) Validate() error {
	if len(c.Tasks) == 0 {
		return errors.New("at least one task must be specified")
	}

	return nil
}
