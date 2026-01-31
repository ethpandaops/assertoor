package runtaskbackground

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ForegroundTask *helper.RawMessageMasked `yaml:"foregroundTask" json:"foregroundTask"`
	BackgroundTask *helper.RawMessageMasked `yaml:"backgroundTask" json:"backgroundTask"`

	ExitOnForegroundSuccess bool `yaml:"exitOnForegroundSuccess" json:"exitOnForegroundSuccess"`
	ExitOnForegroundFailure bool `yaml:"exitOnForegroundFailure" json:"exitOnForegroundFailure"`

	// action when background task stops
	// "ignore" - do nothing (default)
	// "fail" - exit with failure
	// "succeed" - exit with success
	// "failOrIgnore" - exit with failure if background task failed, ignore on success
	OnBackgroundComplete string `yaml:"onBackgroundComplete" json:"onBackgroundComplete"`

	NewVariableScope bool `yaml:"newVariableScope" json:"newVariableScope"`
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
