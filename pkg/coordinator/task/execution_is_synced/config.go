package executionissynced

import "errors"

type Config struct {
	Percent                 float64 `yaml:"percent" json:"percent"`
	WaitForChainProgression bool    `yaml:"wait_for_chain_progression" json:"wait_for_chain_progression"`
	MinBlockHeight          int     `yaml:"min_block_height" json:"min_block_height"`
}

func (c *Config) Validate() error {
	if c.Percent > 100 {
		return errors.New("percent must be less than 100")
	}

	if c.MinBlockHeight < 0 {
		return errors.New("min_block_height must be greater than 0")
	}

	return nil
}

func DefaultConfig() Config {
	return Config{
		Percent:                 100,
		WaitForChainProgression: true,
		MinBlockHeight:          10,
	}
}
