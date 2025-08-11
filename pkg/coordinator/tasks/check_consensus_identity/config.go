package checkconsensusidentity

import (
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
)

type Config struct {
	ClientPattern   string          `yaml:"clientPattern" json:"clientPattern"`
	PollInterval    helper.Duration `yaml:"pollInterval" json:"pollInterval"`
	MinClientCount  int             `yaml:"minClientCount" json:"minClientCount"`
	MaxFailCount    int             `yaml:"maxFailCount" json:"maxFailCount"`
	FailOnCheckMiss bool            `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`

	// CGC (Custody Group Count) checks
	ExpectCGC *uint64 `yaml:"expectCgc" json:"expectCgc"`
	MinCGC    *uint64 `yaml:"minCgc" json:"minCgc"`
	MaxCGC    *uint64 `yaml:"maxCgc" json:"maxCgc"`

	// ENR checks
	ExpectENRField map[string]interface{} `yaml:"expectEnrField" json:"expectEnrField"`

	// PeerID checks
	ExpectPeerIDPattern string `yaml:"expectPeerIdPattern" json:"expectPeerIdPattern"`

	// P2P address checks
	ExpectP2PAddressCount *int   `yaml:"expectP2pAddressCount" json:"expectP2pAddressCount"`
	ExpectP2PAddressMatch string `yaml:"expectP2pAddressMatch" json:"expectP2pAddressMatch"`

	// Metadata checks
	ExpectSeqNumber *uint64 `yaml:"expectSeqNumber" json:"expectSeqNumber"`
	MinSeqNumber    *uint64 `yaml:"minSeqNumber" json:"minSeqNumber"`
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
