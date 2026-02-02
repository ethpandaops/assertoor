package checkclientsarehealthy

import (
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ClientPattern         string          `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for health checking."`
	PollInterval          helper.Duration `yaml:"pollInterval" json:"pollInterval" desc:"Interval between health check polls (e.g., '5s', '1m')."`
	SkipConsensusCheck    bool            `yaml:"skipConsensusCheck" json:"skipConsensusCheck" desc:"If true, skip consensus client health checks."`
	SkipExecutionCheck    bool            `yaml:"skipExecutionCheck" json:"skipExecutionCheck" desc:"If true, skip execution client health checks."`
	ExpectUnhealthy       bool            `yaml:"expectUnhealthy" json:"expectUnhealthy" desc:"If true, expect clients to be unhealthy (inverts success condition)."`
	MinClientCount        int             `yaml:"minClientCount" json:"minClientCount" desc:"Minimum number of healthy clients required."`
	MaxUnhealthyCount     int             `yaml:"maxUnhealthyCount" json:"maxUnhealthyCount" desc:"Maximum number of unhealthy clients allowed (-1 for unlimited)."`
	FailOnCheckMiss       bool            `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when health check condition is not met."`
	ExecutionRPCResultVar string          `yaml:"executionRpcResultVar" json:"executionRpcResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	ConsensusRPCResultVar string          `yaml:"consensusRpcResultVar" json:"consensusRpcResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	ContinueOnPass        bool            `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
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
