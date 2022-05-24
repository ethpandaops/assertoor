package coordinator

import (
	"os"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
	"gopkg.in/yaml.v2"
)

type Consensus struct {
	URL string `yaml:"url"`
}

// ExecutionNode represents a single ethereum execution client.
type Execution struct {
	URL string `yaml:"url"`
}

type Config struct {
	// Test is the name of the test to run.
	Test string
	// Execution is the execution node to use.
	Execution Execution `yaml:"execution"`
	// Consensus is the consensus node to use.
	Consensus Consensus `yaml:"consensus"`
	// Tasks drives the configuration for the individual tasks.
	TaskConfig task.Config `yaml:"task_config"`
}

// DefaultConfig represents a sane-default configuration.
func DefaultConfig() *Config {
	return &Config{
		Test: "both_synced",
		Execution: Execution{
			URL: "http://localhost:8545",
		},
		Consensus: Consensus{
			URL: "http://localhost:5052",
		},
		TaskConfig: task.DefaultConfig(),
	}
}

func NewConfig(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	config := DefaultConfig()

	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}

	return config, nil
}
