package runtasksconcurrent

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Tasks            []helper.RawMessageMasked `yaml:"tasks" json:"tasks" desc:"List of tasks to execute concurrently."`
	NewVariableScope bool                      `yaml:"newVariableScope" json:"newVariableScope" desc:"If true, create a new variable scope for child tasks."`

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
