package checkexecutionsyncstatus

import (
	"errors"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/human-duration"
)

type Config struct {
	ClientNamePatterns      []string       `yaml:"clientNamePatterns" json:"clientNamePatterns"`
	PollInterval            human.Duration `yaml:"pollInterval" json:"pollInterval"`
	ExpectSyncing           bool           `yaml:"expectSyncing" json:"expectSyncing"`
	ExpectMinPercent        float64        `yaml:"expectMinPercent" json:"expectMinPercent"`
	ExpectMaxPercent        float64        `yaml:"expectMaxPercent" json:"expectMaxPercent"`
	MinBlockHeight          int            `yaml:"minBlockHeight" json:"minBlockHeight"`
	WaitForChainProgression bool           `yaml:"waitForChainProgression" json:"waitForChainProgression"`
}

func DefaultConfig() Config {
	return Config{
		ClientNamePatterns: []string{".*"},
		PollInterval:       human.Duration{Duration: 5 * time.Second},
		ExpectMinPercent:   100,
		ExpectMaxPercent:   100,
		MinBlockHeight:     10,
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
