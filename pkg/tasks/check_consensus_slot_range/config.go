package checkconsensusslotrange

import "math"

type Config struct {
	MinSlotNumber  uint64 `yaml:"minSlotNumber" json:"minSlotNumber"`
	MaxSlotNumber  uint64 `yaml:"maxSlotNumber" json:"maxSlotNumber"`
	MinEpochNumber uint64 `yaml:"minEpochNumber" json:"minEpochNumber"`
	MaxEpochNumber uint64 `yaml:"maxEpochNumber" json:"maxEpochNumber"`
	FailIfLower    bool   `yaml:"failIfLower" json:"failIfLower"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if conditions change.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
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
