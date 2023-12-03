package coordinator

import (
	"os"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
	"github.com/ethpandaops/minccino/pkg/coordinator/test"
	web_types "github.com/ethpandaops/minccino/pkg/coordinator/web/types"
	"gopkg.in/yaml.v2"
)

type Config struct {
	// List of execution & consensus clients to use.
	Endpoints []clients.ClientConfig `yaml:"endpoints" json:"endpoints"`

	// WebServer config
	Web *web_types.WebConfig `yaml:"web" json:"web"`

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
