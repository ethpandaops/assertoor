package checkconsensussyncstatus

import (
	"errors"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/human-duration"
)

type Config struct {
	ClientNamePatterns      []string       `yaml:"clientNamePatterns" json:"clientNamePatterns"`
	PollInterval            human.Duration `yaml:"pollInterval" json:"pollInterval"`
	ExpectSyncing           bool           `yaml:"expectSyncing" json:"expectSyncing"`
	ExpectOptimistic        bool           `yaml:"expectOptimistic" json:"expectOptimistic"`
	ExpectMinPercent        float64        `yaml:"expectMinPercent" json:"expectMinPercent"`
	ExpectMaxPercent        float64        `yaml:"expectMaxPercent" json:"expectMaxPercent"`
	MinSlotHeight           int            `yaml:"minSlotHeight" json:"minSlotHeight"`
	WaitForChainProgression bool           `yaml:"waitForChainProgression" json:"waitForChainProgression"`
}

func DefaultConfig() Config {
	return Config{
		ClientNamePatterns: []string{".*"},
		PollInterval:       human.Duration{Duration: 5 * time.Second},
		ExpectMinPercent:   100,
		ExpectMaxPercent:   100,
		MinSlotHeight:      10,
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

	if c.MinSlotHeight < 0 {
		return errors.New("minSlotHeight must be >= 0")
	}

	return nil
}
