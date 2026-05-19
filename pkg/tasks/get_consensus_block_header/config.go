package getconsensusblockheader

import "github.com/ethpandaops/assertoor/pkg/helper"

// Config tells get_consensus_block_header which header to fetch and
// from which client.
type Config struct {
	// ClientPattern selects a single CL client to query. Empty = the
	// first online client from the pool.
	ClientPattern string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern selecting the source CL client. Empty = first online."`

	// Slot, if non-zero, fetches the canonical block at that slot.
	// Mutually exclusive with BlockRoot; takes precedence over the
	// default "head" lookup.
	Slot uint64 `yaml:"slot" json:"slot" desc:"Fetch the canonical block at this slot. Mutually exclusive with blockRoot."`

	// BlockRoot, if non-empty, fetches the block with that root.
	// Accepts 0x-prefixed hex.
	BlockRoot string `yaml:"blockRoot" json:"blockRoot" desc:"Fetch the block with this root (0x-prefixed hex). Mutually exclusive with slot."`

	// HeadOffset, when slot/blockRoot are both unset, fetches the
	// head minus this many slots. Default 0 (= head). Useful for
	// pulling a slightly-back-from-head reference point that's
	// stable across all clients.
	HeadOffset int `yaml:"headOffset" json:"headOffset" desc:"When slot/blockRoot are empty, fetch head - headOffset. Default 0."`

	// MaxLookback bounds the walk-back loop that skips missed slots
	// when fetching by slot. Default 8.
	MaxLookback int `yaml:"maxLookback" json:"maxLookback" desc:"Max consecutive missed slots to skip while resolving slot/headOffset. Default 8."`

	// RequestTimeout caps each beacon-API call. Default 15s.
	RequestTimeout helper.Duration `yaml:"requestTimeout" json:"requestTimeout" desc:"Per-RPC timeout. Default 15s."`
}

func DefaultConfig() Config {
	return Config{
		MaxLookback: 8,
	}
}

func (c *Config) Validate() error {
	if c.HeadOffset < 0 {
		c.HeadOffset = 0
	}

	if c.MaxLookback <= 0 {
		c.MaxLookback = 8
	}

	return nil
}
