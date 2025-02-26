package coordinator

import (
	"os"

	"github.com/noku-team/assertoor/pkg/coordinator/clients"
	"github.com/noku-team/assertoor/pkg/coordinator/db"
	"github.com/noku-team/assertoor/pkg/coordinator/helper"
	"github.com/noku-team/assertoor/pkg/coordinator/names"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	web_types "github.com/noku-team/assertoor/pkg/coordinator/web/types"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Database config
	Database *db.DatabaseConfig `yaml:"database" json:"database"`

	// List of execution & consensus clients to use.
	Endpoints []clients.ClientConfig `yaml:"endpoints" json:"endpoints"`

	// WebServer config
	Web *web_types.WebConfig `yaml:"web" json:"web"`

	// Validator names config
	ValidatorNames *names.Config `yaml:"validatorNames" json:"validatorNames"`

	// Global variables
	GlobalVars map[string]interface{} `yaml:"globalVars" json:"globalVars"`

	// Coordinator config
	Coordinator *CoordinatorConfig `yaml:"coordinator" json:"coordinator"`

	// List of Test configurations.
	Tests []*types.TestConfig `yaml:"tests" json:"tests"`

	// List of yaml files with test configurations
	ExternalTests []*types.ExternalTestConfig `yaml:"externalTests" json:"externalTests"`
}

//nolint:revive // ignore
type CoordinatorConfig struct {
	// Maximum number of tests executed concurrently
	MaxConcurrentTests uint64 `yaml:"maxConcurrentTests" json:"maxConcurrentTests"`

	// Test history cleanup delay
	TestRetentionTime helper.Duration `yaml:"testRetentionTime" json:"testRetentionTime"`
}

// DefaultConfig represents a sane-default configuration.
func DefaultConfig() *Config {
	return &Config{
		Endpoints: []clients.ClientConfig{
			{
				Name:         "local",
				ExecutionURL: "http://localhost:8545",
				ConsensusURL: "http://localhost:5052",
			},
		},
		GlobalVars:    make(map[string]interface{}),
		Coordinator:   &CoordinatorConfig{},
		Tests:         []*types.TestConfig{},
		ExternalTests: []*types.ExternalTestConfig{},
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
