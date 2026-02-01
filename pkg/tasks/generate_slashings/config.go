package generateslashings

import (
	"errors"
)

type Config struct {
	SlashingType         string `yaml:"slashingType" json:"slashingType" desc:"Type of slashing to generate: 'attester' or 'proposer'."`
	LimitPerSlot         int    `yaml:"limitPerSlot" json:"limitPerSlot" desc:"Maximum number of slashing operations to generate per slot."`
	LimitTotal           int    `yaml:"limitTotal" json:"limitTotal" desc:"Total limit on the number of slashing operations to generate."`
	Mnemonic             string `yaml:"mnemonic" json:"mnemonic" desc:"Mnemonic phrase used to generate validator keys."`
	StartIndex           int    `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start generating validator keys."`
	IndexCount           int    `yaml:"indexCount" json:"indexCount" desc:"Number of validator keys to generate from the mnemonic."`
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting operations."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitInclusion       bool   `yaml:"awaitInclusion" json:"awaitInclusion" desc:"Wait for slashings to be included in beacon blocks before completing."`
}

func DefaultConfig() Config {
	return Config{
		SlashingType: "attester",
	}
}

func (c *Config) Validate() error {
	if c.LimitPerSlot == 0 && c.LimitTotal == 0 && c.IndexCount == 0 {
		return errors.New("either limitPerSlot or limitTotal or indexCount must be set")
	}

	if c.Mnemonic == "" {
		return errors.New("mnemonic must be set")
	}

	return nil
}
