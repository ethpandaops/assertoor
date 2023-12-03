package coordinator

import (
	"os"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
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
	// List of execution & consensus clients to use.
	Endpoints []clients.ClientConfig `yaml:"endpoints" json:"endpoints"`

	// Test is the test configuration.
	Test test.Config `yaml:"test" json:"test"`
}

// DefaultConfig represents a sane-default configuration.
func DefaultConfig() *Config {
	return &Config{
		Endpoints: []clients.ClientConfig{
			{
				Name:         "local",
				ExecutionUrl: "http://localhost:8545",
				ConsensusUrl: "http://localhost:5052",
			},
		},
		Test: test.BasicSynced(),
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
