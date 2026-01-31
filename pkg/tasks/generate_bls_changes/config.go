package generateblschanges

import (
	"errors"
)

type Config struct {
	LimitPerSlot         int    `yaml:"limitPerSlot" json:"limitPerSlot"`
	LimitTotal           int    `yaml:"limitTotal" json:"limitTotal"`
	Mnemonic             string `yaml:"mnemonic" json:"mnemonic"`
	StartIndex           int    `yaml:"startIndex" json:"startIndex"`
	IndexCount           int    `yaml:"indexCount" json:"indexCount"`
	TargetAddress        string `yaml:"targetAddress" json:"targetAddress"`
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.LimitPerSlot == 0 && c.LimitTotal == 0 && c.IndexCount == 0 {
		return errors.New("either limitPerSlot or limitTotal or indexCount must be set")
	}

	if c.Mnemonic == "" {
		return errors.New("mnemonic must be set")
	}

	if c.TargetAddress == "" {
		return errors.New("targetAddress must be set")
	}

	return nil
}
