package runtasks

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
)

type Config struct {
	Tasks []helper.RawMessageMasked `yaml:"tasks" json:"tasks"`

	StopChildOnResult bool `yaml:"stopChildOnResult" json:"stopChildOnResult"`
	ExpectFailure     bool `yaml:"expectFailure" json:"expectFailure"`
	ContinueOnFailure bool `yaml:"continueOnFailure" json:"continueOnFailure"`
	NewVariableScope  bool `yaml:"newVariableScope" json:"newVariableScope"`
}

func DefaultConfig() Config {
	return Config{
		Tasks:             []helper.RawMessageMasked{},
		StopChildOnResult: true,
	}
}

func (c *Config) Validate() error {
	if len(c.Tasks) == 0 {
		return errors.New("at least one task must be specified")
	}

	return nil
}
