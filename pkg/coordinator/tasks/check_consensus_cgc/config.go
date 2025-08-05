package checkconsensuscgc

import (
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
)

type Config struct {
	ClientPattern         string          `yaml:"clientPattern" json:"clientPattern"`
	PollInterval          helper.Duration `yaml:"pollInterval" json:"pollInterval"`
	ExpectedCGCValue      int             `yaml:"expectedCgcValue" json:"expectedCgcValue"`
	ExpectedNonValidating int             `yaml:"expectedNonValidating" json:"expectedNonValidating"`
	ExpectedValidating    int             `yaml:"expectedValidating" json:"expectedValidating"`
	MinClientCount        int             `yaml:"minClientCount" json:"minClientCount"`
	FailOnCheckMiss       bool            `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`
	ResultVar             string          `yaml:"resultVar" json:"resultVar"`
}

func DefaultConfig() Config {
	return Config{
		PollInterval:          helper.Duration{Duration: 30 * time.Second},
		ExpectedNonValidating: 0x04, // Default for non-validating consensus layer node
		ExpectedValidating:    0x08, // Default for validating consensus layer node
	}
}

func (c *Config) Validate() error {
	if c.ExpectedCGCValue < 0 {
		return fmt.Errorf("expectedCgcValue must be non-negative")
	}
	if c.ExpectedNonValidating < 0 {
		return fmt.Errorf("expectedNonValidating must be non-negative")
	}
	if c.ExpectedValidating < 0 {
		return fmt.Errorf("expectedValidating must be non-negative")
	}
	return nil
}
