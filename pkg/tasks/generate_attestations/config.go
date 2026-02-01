package generateattestations

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Config struct {
	// Key configuration
	Mnemonic   string `yaml:"mnemonic" json:"mnemonic" desc:"Mnemonic phrase used to derive validator keys."`
	StartIndex int    `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start deriving keys."`
	IndexCount int    `yaml:"indexCount" json:"indexCount" desc:"Number of validator keys to use for generating attestations."`

	// Limit configuration
	LimitTotal  int `yaml:"limitTotal" json:"limitTotal" desc:"Total limit on the number of attestations to generate."`
	LimitEpochs int `yaml:"limitEpochs" json:"limitEpochs" desc:"Number of epochs to generate attestations for."`

	// Client selection
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting attestations."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`

	// Advanced settings
	LastEpochAttestations bool   `yaml:"lastEpochAttestations" json:"lastEpochAttestations" desc:"If true, generate attestations referencing the last epoch instead of current."`
	SendAllLastEpoch      bool   `yaml:"sendAllLastEpoch" json:"sendAllLastEpoch" desc:"If true, send all attestations with last epoch data."`
	LateHead              int    `yaml:"lateHead" json:"lateHead" desc:"Number of slots to delay the head vote in attestations."`
	RandomLateHead        string `yaml:"randomLateHead" json:"randomLateHead" desc:"Random late head range in 'min:max' or 'min-max' format."`
	LateHeadClusterSize   int    `yaml:"lateHeadClusterSize" json:"lateHeadClusterSize" desc:"Size of validator clusters that share the same late head delay."`
}

// ParseRandomLateHead parses the RandomLateHead string in "min:max" or "min-max" format.
// Returns minVal, maxVal values and whether random late head is enabled.
func (c *Config) ParseRandomLateHead() (minVal, maxVal int, enabled bool, err error) {
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

	minVal, err = strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid min value in randomLateHead: %w", err)
	}

	maxVal, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid max value in randomLateHead: %w", err)
	}

	if minVal > maxVal {
		return 0, 0, false, fmt.Errorf("min (%d) cannot be greater than max (%d) in randomLateHead", minVal, maxVal)
	}

	return minVal, maxVal, true, nil
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
