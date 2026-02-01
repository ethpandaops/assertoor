package runtaskoptions

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Task             *helper.RawMessageMasked `yaml:"task" json:"task"`
	NewVariableScope bool                     `yaml:"newVariableScope" json:"newVariableScope"`

	// Retry behavior
	RetryOnFailure bool `yaml:"retryOnFailure" json:"retryOnFailure"`
	MaxRetryCount  uint `yaml:"maxRetryCount" json:"maxRetryCount"`

	// Result transformation
	InvertResult  bool `yaml:"invertResult" json:"invertResult"`
	IgnoreResult  bool `yaml:"ignoreResult" json:"ignoreResult"`
	ExpectFailure bool `yaml:"expectFailure" json:"expectFailure"` // Alias for invertResult
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
