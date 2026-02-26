package runtaskoptions

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	Task             *helper.RawMessageMasked `yaml:"task" json:"task" require:"A" desc:"The task to execute with additional options."`
	NewVariableScope bool                     `yaml:"newVariableScope" json:"newVariableScope" desc:"If true, create a new variable scope for the child task."`

	// Retry behavior
	RetryOnFailure bool `yaml:"retryOnFailure" json:"retryOnFailure" desc:"If true, retry the task on failure."`
	MaxRetryCount  uint `yaml:"maxRetryCount" json:"maxRetryCount" desc:"Maximum number of retry attempts."`

	// Result transformation
	InvertResult  bool `yaml:"invertResult" json:"invertResult" desc:"If true, swap success and failure results."`
	IgnoreResult  bool `yaml:"ignoreResult" json:"ignoreResult" desc:"If true, always report success regardless of child task result."`
	IgnoreFailure bool `yaml:"ignoreFailure" json:"ignoreFailure" desc:"If true, ignore the failure of the child task."`
	ExpectFailure bool `yaml:"expectFailure" json:"expectFailure" desc:"Alias for invertResult - expect the task to fail."`
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
