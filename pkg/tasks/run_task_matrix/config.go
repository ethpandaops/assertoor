package runtaskmatrix

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Task         *helper.RawMessageMasked `yaml:"task" json:"task" require:"A" desc:"The task template to execute for each matrix value."`
	MatrixVar    string                   `yaml:"matrixVar" json:"matrixVar" desc:"Variable name to bind each matrix value to during task execution."`
	MatrixValues []any                    `yaml:"matrixValues" json:"matrixValues" desc:"List of values to iterate over, executing the task for each."`

	// Whether to run tasks concurrently (default: false - sequential)
	RunConcurrent bool `yaml:"runConcurrent" json:"runConcurrent" desc:"If true, run all matrix tasks concurrently instead of sequentially."`

	// Threshold behavior:
	// - 0 (default): No threshold - only evaluate result when ALL tasks complete
	// - >0: Set result when threshold is reached (but continue until all complete unless StopOnThreshold=true)
	SuccessThreshold uint64 `yaml:"successThreshold" json:"successThreshold" desc:"Number of successful tasks required to set success result (0 = all must succeed)."`
	FailureThreshold uint64 `yaml:"failureThreshold" json:"failureThreshold" desc:"Number of failed tasks that triggers failure result (0 = no threshold)."`

	// Early termination - if true, stop immediately when a threshold is reached
	// Default: false - always wait for all tasks to complete
	StopOnThreshold bool `yaml:"stopOnThreshold" json:"stopOnThreshold" desc:"If true, stop immediately when a threshold is reached instead of waiting for all tasks."`

	// Result transformation
	InvertResult bool `yaml:"invertResult" json:"invertResult" desc:"If true, swap success and failure results."`
	IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult" desc:"If true, always report success regardless of child task results."`
}

func DefaultConfig() Config {
	return Config{
		FailureThreshold: 1,
		StopOnThreshold:  true,
	}
}

// IsRunConcurrent returns whether the matrix tasks should run concurrently.
func (c *Config) IsRunConcurrent() bool {
	return c.RunConcurrent
}

func (c *Config) Validate() error {
	if c.Task == nil {
		return errors.New("child task must be specified")
	}

	return nil
}
