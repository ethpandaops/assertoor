package consensuscheckpointhasprogressed

import (
	"fmt"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
)

type Config struct {
	Distance       int64                    `yaml:"distance"`
	CheckpointName consensus.CheckpointName `yaml:"checkpoint_name" json:"checkpoint_name"`
}

func DefaultConfig() Config {
	return Config{
		Distance:       3,
		CheckpointName: consensus.Head,
	}
}

func (c *Config) Validate() error {
	if c.Distance < 0 {
		return fmt.Errorf("distance must be >= 0")
	}

	if c.CheckpointName == "" {
		return fmt.Errorf("checkpoint_name must be provided")
	}

	return nil
}
