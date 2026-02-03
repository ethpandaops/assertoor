package getpubkeysfrommnemonic

import "fmt"

type Config struct {
	Mnemonic   string `yaml:"mnemonic" json:"mnemonic" require:"A" desc:"Mnemonic phrase used to derive validator public keys."`
	StartIndex int    `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start deriving keys."`
	Count      int    `yaml:"count" json:"count" desc:"Number of public keys to derive from the mnemonic."`
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
