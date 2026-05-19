package getconsensusproposerduties

import "github.com/ethpandaops/assertoor/pkg/helper"

// Config tells get_consensus_proposer_duties which epoch to fetch
// and which client to ask.
type Config struct {
	// ClientPattern selects a single CL client to query. Empty = the
	// first online client from the pool.
	ClientPattern string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern selecting the source CL client. Empty = first online."`

	// Epoch is the absolute epoch number to fetch duties for. If
	// zero and EpochOffset is also zero we default to the current
	// epoch.
	Epoch uint64 `yaml:"epoch" json:"epoch" desc:"Absolute epoch to fetch proposer duties for. Empty = current epoch + epochOffset."`

	// EpochOffset is added to the current epoch when Epoch is zero.
	// Negative offsets are clamped at zero. Useful for "next epoch
	// duties" without having to compute the absolute number.
	EpochOffset int `yaml:"epochOffset" json:"epochOffset" desc:"Offset from the current epoch when epoch is empty. Default 0."`

	// MaxDuties caps how many entries we surface on the `duties`
	// output. Default 16 — enough for the GLOAS playbook's needs.
	MaxDuties int `yaml:"maxDuties" json:"maxDuties" desc:"Cap on the number of duties surfaced. Default 16."`

	// RequestTimeout caps the RPC call. Default 15s.
	RequestTimeout helper.Duration `yaml:"requestTimeout" json:"requestTimeout" desc:"Per-RPC timeout. Default 15s."`
}

func DefaultConfig() Config {
	return Config{
		MaxDuties: 16,
	}
}

func (c *Config) Validate() error {
	if c.MaxDuties <= 0 {
		c.MaxDuties = 16
	}

	return nil
}
