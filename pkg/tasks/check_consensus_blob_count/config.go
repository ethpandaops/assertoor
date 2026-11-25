package checkconsensusblobcount

import "fmt"

type Config struct {
	CheckLookback        int    `yaml:"checkLookback" json:"checkLookback" desc:"Number of slots to look back when checking for block proposals."`
	BlockCount           int    `yaml:"blockCount" json:"blockCount" desc:"Number of matching blocks required to pass the check."`
	BlobLimit            string `yaml:"blobLimit" json:"blobLimit" desc:"Blob limit a block must reach: 'target' or 'max'."`
	ValidatorNamePattern string `yaml:"validatorNamePattern" json:"validatorNamePattern" desc:"Regex pattern to match validator names."`
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select the execution client that eth_config is read from."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude execution clients from the eth_config lookup."`
	FailOnOverMax        bool   `yaml:"failOnOverMax" json:"failOnOverMax" desc:"If true, fail immediately when a block carries more blobs than the max."`
}

func DefaultConfig() Config {
	return Config{
		CheckLookback: 1,
		BlockCount:    1,
		BlobLimit:     BlobLimitTarget,
		FailOnOverMax: true,
	}
}

func (c *Config) Validate() error {
	if c.BlobLimit != BlobLimitTarget && c.BlobLimit != BlobLimitMax {
		return fmt.Errorf("unknown blobLimit %q, want %q or %q", c.BlobLimit, BlobLimitTarget, BlobLimitMax)
	}

	if c.BlockCount < 1 {
		return fmt.Errorf("blockCount must be at least 1, got %v", c.BlockCount)
	}

	return nil
}
