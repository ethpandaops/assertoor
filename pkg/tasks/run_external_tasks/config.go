package runexternaltasks

import (
	"errors"
)

type Config struct {
	TestFile       string            `yaml:"testFile" json:"testFile" require:"A" desc:"Path to the external test file to execute."`
	TestConfig     map[string]any    `yaml:"testConfig" json:"testConfig" desc:"Configuration values to pass to the external test."`
	TestConfigVars map[string]string `yaml:"testConfigVars" json:"testConfigVars" format:"expressionMap" desc:"Variable mappings for external test configuration."`
	ExpectFailure  bool              `yaml:"expectFailure" json:"expectFailure" desc:"If true, expect the external test to fail (inverts success condition)."`
	IgnoreFailure  bool              `yaml:"ignoreFailure" json:"ignoreFailure" desc:"If true, ignore failures from the external test."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.TestFile == "" {
		return errors.New("testFile is missing")
	}

	return nil
}
