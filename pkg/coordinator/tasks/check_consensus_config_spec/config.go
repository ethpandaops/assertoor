package checkconsensusconfigspec

import (
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
)

type Config struct {
	// ClientPattern is a regex pattern to filter clients
	ClientPattern string `yaml:"clientPattern" json:"clientPattern"`
	// PollInterval is the interval to poll for client updates
	PollInterval helper.Duration `yaml:"pollInterval" json:"pollInterval"`
	// NetworkPreset is the network preset to use (mainnet or minimal only)
	NetworkPreset string `yaml:"networkPreset" json:"networkPreset"`
	// SpecBranch is the git branch to use for fetching specs (defaults to "dev")
	SpecBranch string `yaml:"specBranch" json:"specBranch"`
	// PresetFiles is the list of preset files to fetch and combine (optional override)
	PresetFiles []string `yaml:"presetFiles" json:"presetFiles"`
	// RequiredFields specifies which fields are mandatory in the spec response
	RequiredFields []string `yaml:"requiredFields" json:"requiredFields"`
	// AllowExtraFields determines if extra fields not in the spec are allowed
	AllowExtraFields bool `yaml:"allowExtraFields" json:"allowExtraFields"`
	
	// Internal computed fields (not configurable by user)
	specSource    string
	presetBaseURL string
}

func DefaultConfig() Config {
	return Config{
		ClientPattern:    ".*",
		PollInterval:     helper.Duration{Duration: 10 * time.Second},
		NetworkPreset:    "mainnet",
		SpecBranch:       "dev",
		PresetFiles:      []string{"phase0.yaml", "altair.yaml", "bellatrix.yaml", "capella.yaml", "deneb.yaml", "electra.yaml", "fulu.yaml"},
		RequiredFields:   []string{}, // Will be populated from combined spec
		AllowExtraFields: true,
	}
}

func (c *Config) Validate() error {
	if c.ClientPattern == "" {
		c.ClientPattern = ".*"
	}
	if c.PollInterval.Duration == 0 {
		c.PollInterval.Duration = 10 * time.Second
	}
	if c.NetworkPreset == "" {
		c.NetworkPreset = "mainnet"
	}
	if c.SpecBranch == "" {
		c.SpecBranch = "dev"
	}
	if len(c.PresetFiles) == 0 {
		c.PresetFiles = []string{"phase0.yaml", "altair.yaml", "bellatrix.yaml", "capella.yaml", "deneb.yaml", "electra.yaml", "fulu.yaml"}
	}
	
	// Validate networkPreset - only mainnet or minimal are valid
	if c.NetworkPreset != "mainnet" && c.NetworkPreset != "minimal" {
		return fmt.Errorf("invalid networkPreset '%s': only 'mainnet' or 'minimal' are supported", c.NetworkPreset)
	}
	
	// Compute the URLs based on the network preset and branch
	c.specSource = fmt.Sprintf("https://raw.githubusercontent.com/ethereum/consensus-specs/%s/configs/%s.yaml", c.SpecBranch, c.NetworkPreset)
	
	// For minimal preset, use the correct path
	if c.NetworkPreset == "minimal" {
		c.presetBaseURL = fmt.Sprintf("https://raw.githubusercontent.com/ethereum/consensus-specs/%s/presets/minimal", c.SpecBranch)
	} else {
		// For mainnet, use the standard mainnet presets
		c.presetBaseURL = fmt.Sprintf("https://raw.githubusercontent.com/ethereum/consensus-specs/%s/presets/mainnet", c.SpecBranch)
	}
	
	return nil
}