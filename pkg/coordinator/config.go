package coordinator

import (
	"os"

	"github.com/ethpandaops/minccino/pkg/coordinator/test"
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
	// Execution is the execution node to use.
	Execution Execution `yaml:"execution" json:"execution"`
	// Consensus is the consensus node to use.
	Consensus Consensus `yaml:"consensus" json:"consensus"`
	// Test is the test configuration.
	Test test.Config `yaml:"test" json:"test"`
}

// DefaultConfig represents a sane-default configuration.
func DefaultConfig() *Config {
	return &Config{
		Test: test.BasicSynced(),
		Execution: Execution{
			URL: "http://localhost:8545",
		},
		Consensus: Consensus{
			URL: "http://localhost:5052",
		},
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
