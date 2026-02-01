package checkexecutionsyncstatus

import (
	"errors"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ClientPattern           string          `yaml:"clientPattern" json:"clientPattern"`
	PollInterval            helper.Duration `yaml:"pollInterval" json:"pollInterval"`
	ExpectSyncing           bool            `yaml:"expectSyncing" json:"expectSyncing"`
	ExpectMinPercent        float64         `yaml:"expectMinPercent" json:"expectMinPercent"`
	ExpectMaxPercent        float64         `yaml:"expectMaxPercent" json:"expectMaxPercent"`
	MinBlockHeight          int             `yaml:"minBlockHeight" json:"minBlockHeight"`
	WaitForChainProgression bool            `yaml:"waitForChainProgression" json:"waitForChainProgression"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if sync status changes.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}

func DefaultConfig() Config {
	return Config{
		PollInterval:     helper.Duration{Duration: 5 * time.Second},
		ExpectMinPercent: 100,
		ExpectMaxPercent: 100,
		MinBlockHeight:   10,
	}
}

func (c *Config) Validate() error {
	if c.ExpectMinPercent > 100 {
		return errors.New("expectMinPercent must be less than 100")
	}

	if c.ExpectMaxPercent > 100 {
		return errors.New("ExpectMaxPercent must be less than 100")
	}

	if c.ExpectMaxPercent < c.ExpectMinPercent {
		return errors.New("ExpectMaxPercent must be <= ExpectMinPercent")
	}

	if c.MinBlockHeight < 0 {
		return errors.New("minBlockHeight must be >= 0")
	}

	return nil
}
