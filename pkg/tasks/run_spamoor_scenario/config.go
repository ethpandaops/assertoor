package runspamoorscenario

import (
	"errors"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/spamoor/scenarios"
)

type Config struct {
	ScenarioName string             `yaml:"scenarioName" json:"scenarioName" desc:"Name of the spamoor scenario to run."`
	PrivateKey   string             `yaml:"privateKey" json:"privateKey" desc:"Private key of the root wallet used to fund scenario wallets."`
	ScenarioYAML helper.IRawMessage `yaml:"scenarioYaml" json:"scenarioYaml" desc:"YAML configuration for the scenario (passed as-is to spamoor)."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.ScenarioName == "" {
		return errors.New("scenarioName must be set")
	}

	if scenarios.GetScenario(c.ScenarioName) == nil {
		availableScenarios := scenarios.GetScenarioNames()
		return errors.New("unknown scenario: " + c.ScenarioName + ". Available scenarios: " + formatScenarioList(availableScenarios))
	}

	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}

func formatScenarioList(names []string) string {
	result := ""

	for i, name := range names {
		if i > 0 {
			result += ", "
		}

		result += name
	}

	return result
}
