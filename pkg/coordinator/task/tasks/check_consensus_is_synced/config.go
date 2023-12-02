package checkconsensusissynced

import "errors"

type Config struct {
	Percent                 float64 `yaml:"percent" json:"percent"`
	WaitForChainProgression bool    `yaml:"wait_for_chain_progression" json:"wait_for_chain_progression"`
	MinSlotHeight           int     `yaml:"min_slot_height" json:"min_slot_height"`
}

func DefaultConfig() Config {
	return Config{
		Percent:                 100,
		WaitForChainProgression: true,
		MinSlotHeight:           10,
	}
}

func (c *Config) Validate() error {
	if c.Percent > 100 {
		return errors.New("percent must be less than 100")
	}

	if c.MinSlotHeight < 0 {
		return errors.New("min_slot_height must be greater than 0")
	}

	return nil
}
