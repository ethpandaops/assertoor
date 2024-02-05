package generatedeposits

import (
	"errors"
)

type Config struct {
	LimitPerSlot         int    `yaml:"limitPerSlot" json:"limitPerSlot"`
	LimitTotal           int    `yaml:"limitTotal" json:"limitTotal"`
	Mnemonic             string `yaml:"mnemonic" json:"mnemonic"`
	StartIndex           int    `yaml:"startIndex" json:"startIndex"`
	IndexCount           int    `yaml:"indexCount" json:"indexCount"`
	WalletPrivkey        string `yaml:"walletPrivkey" json:"walletPrivkey"`
	DepositContract      string `yaml:"depositContract" json:"depositContract"`
	DepositTxFeeCap      int64  `yaml:"depositTxFeeCap" json:"depositTxFeeCap"`
	DepositTxTipCap      int64  `yaml:"depositTxTipCap" json:"depositTxTipCap"`
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
}

func DefaultConfig() Config {
	return Config{
		DepositTxFeeCap: 100000000000, // 100 gwei
		DepositTxTipCap: 1000000000,   // 1 gwei
	}
}

func (c *Config) Validate() error {
	if c.LimitPerSlot == 0 && c.LimitTotal == 0 && c.IndexCount == 0 {
		return errors.New("either limitPerSlot or limitTotal or indexCount must be set")
	}

	if c.Mnemonic == "" {
		return errors.New("mnemonic must be set")
	}

	if c.WalletPrivkey == "" {
		return errors.New("walletPrivkey must be set")
	}

	if c.DepositContract == "" {
		return errors.New("depositContract must be set")
	}

	return nil
}
