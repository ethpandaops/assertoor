package getpubkeysfrommnemonic

import "fmt"

type Config struct {
	Mnemonic   string `yaml:"mnemonic" json:"mnemonic"`
	StartIndex int    `yaml:"startIndex" json:"startIndex"`
	Count      int    `yaml:"count" json:"count"`
}

func DefaultConfig() Config {
	return Config{
		Count: 1,
	}
}

func (c *Config) Validate() error {
	if c.Mnemonic == "" {
		return fmt.Errorf("mnemonic is required")
	}

	return nil
}
