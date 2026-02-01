package checkconsensussyncstatus

import (
	"errors"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ClientPattern           string          `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for sync status checking."`
	PollInterval            helper.Duration `yaml:"pollInterval" json:"pollInterval" desc:"Interval between sync status polls (e.g., '5s', '1m')."`
	ExpectSyncing           bool            `yaml:"expectSyncing" json:"expectSyncing" desc:"If true, expect clients to be syncing."`
	ExpectOptimistic        bool            `yaml:"expectOptimistic" json:"expectOptimistic" desc:"If true, expect clients to be in optimistic mode."`
	ExpectMinPercent        float64         `yaml:"expectMinPercent" json:"expectMinPercent" desc:"Minimum percentage of clients expected to match the sync condition."`
	ExpectMaxPercent        float64         `yaml:"expectMaxPercent" json:"expectMaxPercent" desc:"Maximum percentage of clients expected to match the sync condition."`
	MinSlotHeight           int             `yaml:"minSlotHeight" json:"minSlotHeight" desc:"Minimum slot height required before checking sync status."`
	WaitForChainProgression bool            `yaml:"waitForChainProgression" json:"waitForChainProgression" desc:"If true, wait for the chain to progress before checking."`
	ContinueOnPass          bool            `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
}

func DefaultConfig() Config {
	return Config{
		PollInterval:     helper.Duration{Duration: 5 * time.Second},
		ExpectMinPercent: 100,
		ExpectMaxPercent: 100,
		MinSlotHeight:    10,
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
