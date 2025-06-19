package runtaskmatrix

import (
	"errors"

	"github.com/erigontech/assertoor/pkg/coordinator/helper"
)

type Config struct {
	// matrix variable name
	RunConcurrent bool `yaml:"runConcurrent" json:"runConcurrent"`

	// number of successful child tasks to make this task succeed (0 = all tasks)
	SucceedTaskCount uint64 `yaml:"succeedTaskCount" json:"succeedTaskCount"`

	// number of failed child tasks to make this task fail (0 = all tasks)
	FailTaskCount uint64 `yaml:"failTaskCount" json:"failTaskCount"`

	// fail task if neither succeedTaskCount nor failTaskCount is reached, but all tasks completed
	FailOnUndecided bool `yaml:"failOnUndecided" json:"failOnUndecided"`

	// matrix variable name
	MatrixValues []interface{} `yaml:"matrixValues" json:"matrixValues"`

	// matrix variable name
	MatrixVar string `yaml:"matrixVar" json:"matrixVar"`

	// child task
	Task *helper.RawMessageMasked `yaml:"task" json:"task"`
}

func DefaultConfig() Config {
	return Config{
		FailOnUndecided: true,
	}
}

func (c *Config) Validate() error {
	if c.Task == nil {
		return errors.New("child task must be specified")
	}

	return nil
}
