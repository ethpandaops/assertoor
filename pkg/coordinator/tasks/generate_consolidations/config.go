package generateconsolidations

import (
	"errors"
)

type Config struct {
	LimitPerSlot         int    `yaml:"limitPerSlot" json:"limitPerSlot"`
	LimitTotal           int    `yaml:"limitTotal" json:"limitTotal"`
	SourceMnemonic       string `yaml:"sourceMnemonic" json:"sourceMnemonic"`
	TargetMnemonic       string `yaml:"targetMnemonic" json:"targetMnemonic"`
	SourceStartIndex     int    `yaml:"sourceStartIndex" json:"sourceStartIndex"`
	SourceIndexCount     int    `yaml:"sourceIndexCount" json:"sourceIndexCount"`
	TargetValidator      uint64 `yaml:"targetValidator" json:"targetValidator"`
	TargetKeyIndex       uint64 `yaml:"targetIndex" json:"targetIndex"`
	ConsolidationEpoch   uint64 `yaml:"consolidationEpoch" json:"consolidationEpoch"`
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.LimitPerSlot == 0 && c.LimitTotal == 0 && c.SourceIndexCount == 0 {
		return errors.New("either limitPerSlot or limitTotal or indexCount must be set")
	}

	if c.SourceMnemonic == "" {
		return errors.New("sourceMnemonic must be set")
	}

	if c.TargetMnemonic == "" {
		return errors.New("sourceMnemonic must be set")
	}

	return nil
}
