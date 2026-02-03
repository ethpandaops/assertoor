package checkconsensusslotrange

import "math"

type Config struct {
	MinSlotNumber  uint64 `yaml:"minSlotNumber" json:"minSlotNumber" desc:"Minimum slot number required to pass the check."`
	MaxSlotNumber  uint64 `yaml:"maxSlotNumber" json:"maxSlotNumber" desc:"Maximum slot number allowed to pass the check."`
	MinEpochNumber uint64 `yaml:"minEpochNumber" json:"minEpochNumber" desc:"Minimum epoch number required to pass the check."`
	MaxEpochNumber uint64 `yaml:"maxEpochNumber" json:"maxEpochNumber" desc:"Maximum epoch number allowed to pass the check."`
	FailIfLower    bool   `yaml:"failIfLower" json:"failIfLower" desc:"If true, fail immediately when slot/epoch is below minimum instead of waiting."`
	ContinueOnPass bool   `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
}

func DefaultConfig() Config {
	return Config{
		MaxSlotNumber:  math.MaxUint64,
		MaxEpochNumber: math.MaxUint64,
	}
}

func (c *Config) Validate() error {
	return nil
}
