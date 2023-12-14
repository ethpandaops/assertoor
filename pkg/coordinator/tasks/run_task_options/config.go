package runtaskoptions

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
)

type Config struct {
	Task             *helper.RawMessage `yaml:"task" json:"tasks"`
	ExitOnResult     bool               `yaml:"exitOnResult" json:"exitOnResult"`
	InvertResult     bool               `yaml:"invertResult" json:"invertResult"`
	ExpectFailure    bool               `yaml:"expectFailure" json:"expectFailure"`
	IgnoreFailure    bool               `yaml:"ignoreFailure" json:"ignoreFailure"`
	NewVariableScope bool               `yaml:"newVariableScope" json:"newVariableScope"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.Task == nil {
		return errors.New("child task must be specified")
	}

	return nil
}
