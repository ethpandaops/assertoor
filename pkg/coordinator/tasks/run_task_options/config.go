package runtaskoptions

import (
	"errors"

	"github.com/noku-team/assertoor/pkg/coordinator/helper"
)

type Config struct {
	Task *helper.RawMessageMasked `yaml:"task" json:"tasks"`

	PropagateResult  bool `yaml:"propagateResult" json:"propagateResult"`
	ExitOnResult     bool `yaml:"exitOnResult" json:"exitOnResult"`
	InvertResult     bool `yaml:"invertResult" json:"invertResult"`
	ExpectFailure    bool `yaml:"expectFailure" json:"expectFailure"`
	IgnoreFailure    bool `yaml:"ignoreFailure" json:"ignoreFailure"`
	RetryOnFailure   bool `yaml:"retryOnFailure" json:"retryOnFailure"`
	MaxRetryCount    uint `yaml:"maxRetryCount" json:"maxRetryCount"`
	NewVariableScope bool `yaml:"newVariableScope" json:"newVariableScope"`
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
