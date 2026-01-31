package checkconsensusslotrange

import "math"

type Config struct {
	MinSlotNumber  uint64 `yaml:"minSlotNumber" json:"minSlotNumber"`
	MaxSlotNumber  uint64 `yaml:"maxSlotNumber" json:"maxSlotNumber"`
	MinEpochNumber uint64 `yaml:"minEpochNumber" json:"minEpochNumber"`
	MaxEpochNumber uint64 `yaml:"maxEpochNumber" json:"maxEpochNumber"`
	FailIfLower    bool   `yaml:"failIfLower" json:"failIfLower"`
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
