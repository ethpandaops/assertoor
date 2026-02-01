package runtasks

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Tasks            []helper.RawMessageMasked `yaml:"tasks" json:"tasks"`
	NewVariableScope bool                      `yaml:"newVariableScope" json:"newVariableScope"`

	// Failure handling (default: stop on first failure)
	ContinueOnFailure bool `yaml:"continueOnFailure" json:"continueOnFailure"`

	// Result transformation
	InvertResult bool `yaml:"invertResult" json:"invertResult"` // Swap success/failure
	IgnoreResult bool `yaml:"ignoreResult" json:"ignoreResult"` // Always succeed
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
