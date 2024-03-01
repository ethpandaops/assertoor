package checkexecutionconversionstate

import (
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/human-duration"
)

type Config struct {
	ClientPattern    string         `yaml:"clientPattern" json:"clientPattern"`
	PollInterval     human.Duration `yaml:"pollInterval" json:"pollInterval"`
	ExpectStarted    bool           `yaml:"expectStarted" json:"expectStarted"`
	ExpectFinished   bool           `yaml:"expectFinished" json:"expectFinished"`
	FailOnUnexpected bool           `yaml:"failOnUnexpected" json:"failOnUnexpected"`
}

func DefaultConfig() Config {
	return Config{
		PollInterval: human.Duration{Duration: 5 * time.Second},
	}
}

func (c *Config) Validate() error {
	return nil
}
