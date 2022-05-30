package botharesynced

import (
	consensusissyncing "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_syncing"
	executionissyncing "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_is_syncing"
)

type Config struct {
	ConsensusissyncingConfig consensusissyncing.Config `yaml:"consensus" json:"consensus"`
	ExecutionissyncingConfig executionissyncing.Config `yaml:"execution" json:"execution"`
}

func DefaultConfig() Config {
	return Config{
		ConsensusissyncingConfig: consensusissyncing.DefaultConfig(),
		ExecutionissyncingConfig: executionissyncing.DefaultConfig(),
	}
}

func (c *Config) Validate() error {
	if err := c.ConsensusissyncingConfig.Validate(); err != nil {
		return err
	}

	if err := c.ExecutionissyncingConfig.Validate(); err != nil {
		return err
	}

	return nil
}
