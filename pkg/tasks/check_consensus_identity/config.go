package checkconsensusidentity

import (
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	ClientPattern   string          `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for identity checking."`
	PollInterval    helper.Duration `yaml:"pollInterval" json:"pollInterval" desc:"Interval between identity check polls (e.g., '10s', '1m')."`
	MinClientCount  int             `yaml:"minClientCount" json:"minClientCount" desc:"Minimum number of clients required to pass the check."`
	MaxFailCount    int             `yaml:"maxFailCount" json:"maxFailCount" desc:"Maximum number of clients allowed to fail the check (-1 for unlimited)."`
	FailOnCheckMiss bool            `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when identity check condition is not met."`

	// CGC (Custody Group Count) checks
	ExpectCGC *uint64 `yaml:"expectCgc" json:"expectCgc" desc:"Expected custody group count (CGC) value."`
	MinCGC    *uint64 `yaml:"minCgc" json:"minCgc" desc:"Minimum custody group count required."`
	MaxCGC    *uint64 `yaml:"maxCgc" json:"maxCgc" desc:"Maximum custody group count allowed."`

	// ENR checks
	ExpectENRField map[string]interface{} `yaml:"expectEnrField" json:"expectEnrField" desc:"Map of ENR field names to expected values."`

	// PeerID checks
	ExpectPeerIDPattern string `yaml:"expectPeerIdPattern" json:"expectPeerIdPattern" desc:"Regex pattern that peer ID must match."`

	// P2P address checks
	ExpectP2PAddressCount *int   `yaml:"expectP2pAddressCount" json:"expectP2pAddressCount" desc:"Expected number of P2P addresses."`
	ExpectP2PAddressMatch string `yaml:"expectP2pAddressMatch" json:"expectP2pAddressMatch" desc:"Regex pattern that P2P addresses must match."`

	// Metadata checks
	ExpectSeqNumber *uint64 `yaml:"expectSeqNumber" json:"expectSeqNumber" desc:"Expected metadata sequence number."`
	MinSeqNumber    *uint64 `yaml:"minSeqNumber" json:"minSeqNumber" desc:"Minimum metadata sequence number required."`
	ContinueOnPass  bool    `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
}

func DefaultConfig() Config {
	return Config{
		PollInterval:   helper.Duration{Duration: 10 * time.Second},
		MaxFailCount:   -1,
		MinClientCount: 1,
	}
}

func (c *Config) Validate() error {
	if c.ClientPattern == "" {
		return fmt.Errorf("clientPattern is required")
	}

	if c.MinCGC != nil && c.MaxCGC != nil && *c.MinCGC > *c.MaxCGC {
		return fmt.Errorf("minCgc must be <= maxCgc")
	}

	if c.ExpectP2PAddressCount != nil && *c.ExpectP2PAddressCount < 0 {
		return fmt.Errorf("expectP2pAddressCount must be >= 0")
	}

	return nil
}
