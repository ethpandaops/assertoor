package runtaskmatrix

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Task         *helper.RawMessageMasked `yaml:"task" json:"task"`
	MatrixVar    string                   `yaml:"matrixVar" json:"matrixVar"`
	MatrixValues []any                    `yaml:"matrixValues" json:"matrixValues"`

	// Whether to run tasks concurrently (default: false - sequential)
	RunConcurrent bool `yaml:"runConcurrent" json:"runConcurrent"`

	// Threshold behavior:
	// - 0 (default): No threshold - only evaluate result when ALL tasks complete
	// - >0: Set result when threshold is reached (but continue until all complete unless StopOnThreshold=true)
	SuccessThreshold uint64 `yaml:"successThreshold" json:"successThreshold"`
	FailureThreshold uint64 `yaml:"failureThreshold" json:"failureThreshold"`

	// Early termination - if true, stop immediately when a threshold is reached
	// Default: false - always wait for all tasks to complete
	StopOnThreshold bool `yaml:"stopOnThreshold" json:"stopOnThreshold"`

	// Result transformation
	InvertResult bool `yaml:"invertResult" json:"invertResult"`
	IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult"`
}

func DefaultConfig() Config {
	return Config{
		FailureThreshold: 1,
		StopOnThreshold:  true,
	}
}

func (c *Config) Validate() error {
	if c.Task == nil {
		return errors.New("child task must be specified")
	}

	return nil
}
