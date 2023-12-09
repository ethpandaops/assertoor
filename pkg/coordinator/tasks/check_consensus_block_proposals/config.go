package checkconsensusblockproposals

import (
	"errors"
)

type Config struct {
	BlockCount          int    `yaml:"blockCount" json:"blockCount"`
	GraffitiPattern     string `yaml:"graffitiPattern" json:"graffitiPattern"`
	MinDepositCount     int    `yaml:"minDepositCount" json:"minDepositCount"`
	MinExitCount        int    `yaml:"minExitCount" json:"minExitCount"`
	MinSlashingCount    int    `yaml:"minSlashingCount" json:"minSlashingCount"`
	MinBlsChangeCount   int    `yaml:"minBlsChangeCount" json:"minBlsChangeCount"`
	MinWithdrawalCount  int    `yaml:"minWithdrawalCount" json:"minWithdrawalCount"`
	MinTransactionCount int    `yaml:"minTransactionCount" json:"minTransactionCount"`
	MinBlobCount        int    `yaml:"minBlobCount" json:"minBlobCount"`
}

func DefaultConfig() Config {
	return Config{
		BlockCount: 1,
	}
}

func (c *Config) Validate() error {
	if c.BlockCount <= 0 {
		return errors.New("blockCount must be bigger than 0")
	}

	return nil
}
