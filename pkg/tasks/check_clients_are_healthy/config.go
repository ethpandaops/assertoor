package checkclientsarehealthy

import (
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ClientPattern         string          `yaml:"clientPattern" json:"clientPattern"`
	PollInterval          helper.Duration `yaml:"pollInterval" json:"pollInterval"`
	SkipConsensusCheck    bool            `yaml:"skipConsensusCheck" json:"skipConsensusCheck"`
	SkipExecutionCheck    bool            `yaml:"skipExecutionCheck" json:"skipExecutionCheck"`
	ExpectUnhealthy       bool            `yaml:"expectUnhealthy" json:"expectUnhealthy"`
	MinClientCount        int             `yaml:"minClientCount" json:"minClientCount"`
	MaxUnhealthyCount     int             `yaml:"maxUnhealthyCount" json:"maxUnhealthyCount"`
	FailOnCheckMiss       bool            `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`
	ExecutionRPCResultVar string          `yaml:"executionRpcResultVar" json:"executionRpcResultVar"`
	ConsensusRPCResultVar string          `yaml:"consensusRpcResultVar" json:"consensusRpcResultVar"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if clients become unhealthy.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}

func DefaultConfig() Config {
	return Config{
		PollInterval:      helper.Duration{Duration: 5 * time.Second},
		MaxUnhealthyCount: -1,
	}
}

func (c *Config) Validate() error {
	return nil
}
