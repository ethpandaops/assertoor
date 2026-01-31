package runexternaltasks

import (
	"errors"
)

type Config struct {
	TestFile       string            `yaml:"testFile" json:"testFile"`
	TestConfig     map[string]any    `yaml:"testConfig" json:"testConfig"`
	TestConfigVars map[string]string `yaml:"testConfigVars" json:"testConfigVars"`
	ExpectFailure  bool              `yaml:"expectFailure" json:"expectFailure"`
	IgnoreFailure  bool              `yaml:"ignoreFailure" json:"ignoreFailure"`
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
