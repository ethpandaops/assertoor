package generateattestations

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	// Key configuration
	Mnemonic   string `yaml:"mnemonic" json:"mnemonic"`
	StartIndex int    `yaml:"startIndex" json:"startIndex"`
	IndexCount int    `yaml:"indexCount" json:"indexCount"`

	// Limit configuration
	LimitTotal  int `yaml:"limitTotal" json:"limitTotal"`
	LimitEpochs int `yaml:"limitEpochs" json:"limitEpochs"`

	// Client selection
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`

	// Advanced settings
	LastEpochAttestations bool   `yaml:"lastEpochAttestations" json:"lastEpochAttestations"`
	SendAllLastEpoch      bool   `yaml:"sendAllLastEpoch" json:"sendAllLastEpoch"`
	LateHead              int    `yaml:"lateHead" json:"lateHead"`
	RandomLateHead        string `yaml:"randomLateHead" json:"randomLateHead"`
	LateHeadClusterSize   int    `yaml:"lateHeadClusterSize" json:"lateHeadClusterSize"`
}

// ParseRandomLateHead parses the RandomLateHead string in "min:max" or "min-max" format.
// Returns min, max values and whether random late head is enabled.
func (c *Config) ParseRandomLateHead() (min, max int, enabled bool, err error) {
	if c.RandomLateHead == "" {
		return 0, 0, false, nil
	}

	// Try colon separator first, then dash
	var parts []string
	if strings.Contains(c.RandomLateHead, ":") {
		parts = strings.Split(c.RandomLateHead, ":")
	} else if strings.Contains(c.RandomLateHead, "-") {
		parts = strings.Split(c.RandomLateHead, "-")
	}

	if len(parts) != 2 {
		return 0, 0, false, fmt.Errorf("randomLateHead must be in 'min:max' or 'min-max' format, got: %s", c.RandomLateHead)
	}

	min, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid min value in randomLateHead: %w", err)
	}

	max, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid max value in randomLateHead: %w", err)
	}

	if min > max {
		return 0, 0, false, fmt.Errorf("min (%d) cannot be greater than max (%d) in randomLateHead", min, max)
	}

	return min, max, true, nil
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.LimitTotal == 0 && c.LimitEpochs == 0 {
		return errors.New("either limitTotal or limitEpochs must be set")
	}

	if c.Mnemonic == "" {
		return errors.New("mnemonic must be set")
	}

	if c.IndexCount == 0 {
		return errors.New("indexCount must be set")
	}

	if c.RandomLateHead != "" {
		if _, _, _, err := c.ParseRandomLateHead(); err != nil {
			return err
		}
	}

	return nil
}
