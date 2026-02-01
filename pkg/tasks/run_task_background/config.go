package runtaskbackground

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ForegroundTask   *helper.RawMessageMasked `yaml:"foregroundTask" json:"foregroundTask"`
	BackgroundTask   *helper.RawMessageMasked `yaml:"backgroundTask" json:"backgroundTask"`
	NewVariableScope bool                     `yaml:"newVariableScope" json:"newVariableScope"`

	// When to complete (based on foreground task result)
	// These allow early exit even if foreground task hasn't returned yet
	ExitOnForegroundSuccess bool `yaml:"exitOnForegroundSuccess" json:"exitOnForegroundSuccess"`
	ExitOnForegroundFailure bool `yaml:"exitOnForegroundFailure" json:"exitOnForegroundFailure"`

	// What happens if background task completes first
	// "ignore" (default) - do nothing
	// "fail" - exit with failure
	// "succeed" / "success" - exit with success
	// "failOrIgnore" - exit with failure if background task failed, ignore on success
	OnBackgroundComplete string `yaml:"onBackgroundComplete" json:"onBackgroundComplete"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.ForegroundTask == nil {
		return errors.New("foreground task must be specified")
	}

	return nil
}
