package assertoor

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/test"
	"github.com/ethpandaops/assertoor/pkg/types"
	web_types "github.com/ethpandaops/assertoor/pkg/web/types"
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
	GlobalVars map[string]any `yaml:"globalVars" json:"globalVars"`

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
		GlobalVars:    make(map[string]any),
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

func (c *Config) Validate() error {
	var errs []error

	// Validate database config
	if c.Database != nil {
		if c.Database.Engine != "" && c.Database.Engine != "sqlite" && c.Database.Engine != "postgres" {
			errs = append(errs, fmt.Errorf("invalid database engine: %s", c.Database.Engine))
		}
	}

	// Validate endpoints
	for i, endpoint := range c.Endpoints {
		if endpoint.Name == "" {
			errs = append(errs, fmt.Errorf("endpoint[%d]: name cannot be empty", i))
		}

		if endpoint.ConsensusURL == "" && endpoint.ExecutionURL == "" {
			errs = append(errs, fmt.Errorf("endpoint[%d] '%s': must have at least one URL", i, endpoint.Name))
		}
		// Validate URLs are parseable
		if endpoint.ConsensusURL != "" {
			if _, err := url.Parse(endpoint.ConsensusURL); err != nil {
				errs = append(errs, fmt.Errorf("endpoint[%d] '%s': invalid consensus URL: %v", i, endpoint.Name, err))
			}
		}

		if endpoint.ExecutionURL != "" {
			if _, err := url.Parse(endpoint.ExecutionURL); err != nil {
				errs = append(errs, fmt.Errorf("endpoint[%d] '%s': invalid execution URL: %v", i, endpoint.Name, err))
			}
		}
	}

	// Validate web config
	if c.Web != nil {
		if c.Web.Frontend != nil && c.Web.Frontend.Enabled {
			// Validate port is in valid range
			if c.Web.Server.Port != "" {
				if port, err := strconv.Atoi(c.Web.Server.Port); err != nil {
					errs = append(errs, fmt.Errorf("invalid web server port: %s (must be a number)", c.Web.Server.Port))
				} else if port < 1 || port > 65535 {
					errs = append(errs, fmt.Errorf("invalid web server port: %d (must be between 1 and 65535)", port))
				}
			}
		}
	}

	// Validate coordinator config
	if c.Coordinator != nil {
		if err := c.Coordinator.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("coordinator config: %v", err))
		}
	}

	// Validate tests
	for i, testCfg := range c.Tests {
		if testCfg.ID == "" {
			errs = append(errs, fmt.Errorf("test[%d]: ID cannot be empty", i))
		}

		if testCfg.Name == "" {
			errs = append(errs, fmt.Errorf("test[%d] '%s': name cannot be empty", i, testCfg.ID))
		}

		// Validate task configurations
		if err := test.ValidateTestConfig(testCfg); err != nil {
			errs = append(errs, fmt.Errorf("test[%d] '%s': %v", i, testCfg.ID, err))
		}
	}

	// Validate external tests
	for i, extTest := range c.ExternalTests {
		if extTest.File == "" {
			errs = append(errs, fmt.Errorf("external test[%d]: file cannot be empty", i))
		}

		if extTest.ID == "" {
			errs = append(errs, fmt.Errorf("external test[%d]: ID cannot be empty", i))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n%s", formatErrors(errs))
	}

	return nil
}

func formatErrors(errs []error) string {
	var buf strings.Builder
	for _, err := range errs {
		buf.WriteString("  - ")
		buf.WriteString(err.Error())
		buf.WriteString("\n")
	}

	return strings.TrimSuffix(buf.String(), "\n")
}

func (c *CoordinatorConfig) Validate() error {
	if c.TestRetentionTime.Duration != 0 {
		// Duration is valid if it parsed successfully
		if c.TestRetentionTime.Duration < 0 {
			return fmt.Errorf("testRetentionTime cannot be negative")
		}
	}

	return nil
}
