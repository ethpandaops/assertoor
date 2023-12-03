package checkclientsarehealthy

import (
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/human-duration"
)

type Config struct {
	ClientNamePatterns []string       `yaml:"clientNamePatterns" json:"clientNamePatterns"`
	PollInterval       human.Duration `yaml:"pollInterval" json:"pollInterval"`
	SkipConsensusCheck bool           `yaml:"skipConsensusCheck" json:"skipConsensusCheck"`
	SkipExecutionCheck bool           `yaml:"skipExecutionCheck" json:"skipExecutionCheck"`
	ExpectUnhealthy    bool           `yaml:"expectUnhealthy" json:"expectUnhealthy"`
}

func DefaultConfig() Config {
	return Config{
		ClientNamePatterns: []string{".*"},
		PollInterval:       human.Duration{Duration: 5 * time.Second},
	}
}

func (c *Config) Validate() error {
	return nil
}
