package runtaskbackground

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ForegroundTask   *helper.RawMessageMasked `yaml:"foregroundTask" json:"foregroundTask" desc:"The primary task to execute in the foreground."`
	BackgroundTask   *helper.RawMessageMasked `yaml:"backgroundTask" json:"backgroundTask" desc:"The task to execute in the background while foreground runs."`
	NewVariableScope bool                     `yaml:"newVariableScope" json:"newVariableScope" desc:"If true, create a new variable scope for child tasks."`

	// When to complete (based on foreground task result)
	// These allow early exit even if foreground task hasn't returned yet
	ExitOnForegroundSuccess bool `yaml:"exitOnForegroundSuccess" json:"exitOnForegroundSuccess" desc:"If true, exit immediately when foreground task succeeds."`
	ExitOnForegroundFailure bool `yaml:"exitOnForegroundFailure" json:"exitOnForegroundFailure" desc:"If true, exit immediately when foreground task fails."`

	// What happens if background task completes first
	// "ignore" (default) - do nothing
	// "fail" - exit with failure
	// "succeed" / "success" - exit with success
	// "failOrIgnore" - exit with failure if background task failed, ignore on success
	OnBackgroundComplete string `yaml:"onBackgroundComplete" json:"onBackgroundComplete" desc:"Action when background task completes: 'ignore', 'fail', 'succeed', or 'failOrIgnore'."`
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
