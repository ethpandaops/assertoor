package runtasks

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Tasks            []helper.RawMessageMasked `yaml:"tasks" json:"tasks" require:"A" desc:"List of tasks to execute sequentially."`
	NewVariableScope bool                      `yaml:"newVariableScope" json:"newVariableScope" desc:"If true, create a new variable scope for child tasks."`

	// Failure handling (default: stop on first failure)
	ContinueOnFailure bool `yaml:"continueOnFailure" json:"continueOnFailure" desc:"If true, continue executing remaining tasks even if one fails."`

	// Result transformation
	InvertResult bool `yaml:"invertResult" json:"invertResult" desc:"If true, swap success and failure results."`
	IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult" desc:"If true, always report success regardless of child task results."`
}

func DefaultConfig() Config {
	return Config{
		Tasks: []helper.RawMessageMasked{},
	}
}

func (c *Config) Validate() error {
	if len(c.Tasks) == 0 {
		return errors.New("at least one task must be specified")
	}

	return nil
}
