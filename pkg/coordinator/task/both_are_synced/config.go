package botharesynced

import (
	consensusissynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_synced"
	executionissynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_is_synced"
)

type Config struct {
	ConsensusIsSyncedConfig consensusissynced.Config `yaml:"consensus" json:"consensus"`
	ExecutionIsSyncedConfig executionissynced.Config `yaml:"execution" json:"execution"`
}

func DefaultConfig() Config {
	return Config{
		ConsensusIsSyncedConfig: consensusissynced.DefaultConfig(),
		ExecutionIsSyncedConfig: executionissynced.DefaultConfig(),
	}
}

func (c *Config) Validate() error {
	if err := c.ConsensusIsSyncedConfig.Validate(); err != nil {
		return err
	}

	if err := c.ExecutionIsSyncedConfig.Validate(); err != nil {
		return err
	}

	return nil
}
