package generateexits

import (
	"errors"
)

type Config struct {
	LimitPerSlot         int    `yaml:"limitPerSlot" json:"limitPerSlot" require:"A.1" desc:"Maximum number of exit operations to generate per slot."`
	LimitTotal           int    `yaml:"limitTotal" json:"limitTotal" require:"A.2" desc:"Total limit on the number of exit operations to generate."`
	Mnemonic             string `yaml:"mnemonic" json:"mnemonic" require:"B" desc:"Mnemonic phrase used to generate validator keys."`
	StartIndex           int    `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start generating validator keys."`
	IndexCount           int    `yaml:"indexCount" json:"indexCount" require:"A.3" desc:"Number of validator keys to generate from the mnemonic."`
	ExitEpoch            int64  `yaml:"exitEpoch" json:"exitEpoch" desc:"Exit epoch to set in the voluntary exit message (-1 for current epoch)."`
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting operations."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitInclusion       bool   `yaml:"awaitInclusion" json:"awaitInclusion" desc:"Wait for voluntary exits to be included in beacon blocks before completing."`
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

	return nil
}
