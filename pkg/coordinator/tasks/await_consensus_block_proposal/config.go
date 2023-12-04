package awaitconsensusblockproposal

import (
	"errors"
)

type Config struct {
	BlockCount       int      `yaml:"blockCount" json:"blockCount"`
	GraffitiPatterns []string `yaml:"graffitiPatterns" json:"graffitiPatterns"`
}

func DefaultConfig() Config {
	return Config{
		BlockCount: 1,
	}
}

func (c *Config) Validate() error {
	if c.BlockCount <= 0 {
		return errors.New("blockCount must be bigger than 0")
	}

	return nil
}
