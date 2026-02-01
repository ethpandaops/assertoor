package runtasksconcurrent

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Tasks            []helper.RawMessageMasked `yaml:"tasks" json:"tasks"`
	NewVariableScope bool                      `yaml:"newVariableScope" json:"newVariableScope"`

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
		Tasks:            []helper.RawMessageMasked{},
		FailureThreshold: 1,
		StopOnThreshold:  true,
		NewVariableScope: true,
	}
}

func (c *Config) Validate() error {
	if len(c.Tasks) == 0 {
		return errors.New("at least one task must be specified")
	}

	return nil
}
