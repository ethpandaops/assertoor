package generatewithdrawalrequests

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerSlot              int      `yaml:"limitPerSlot" json:"limitPerSlot"`
	LimitTotal                int      `yaml:"limitTotal" json:"limitTotal"`
	LimitPending              int      `yaml:"limitPending" json:"limitPending"`
	SourcePubkey              string   `yaml:"sourcePubkey" json:"sourcePubkey"`
	SourceMnemonic            string   `yaml:"sourceMnemonic" json:"sourceMnemonic"`
	SourceStartIndex          int      `yaml:"sourceStartIndex" json:"sourceStartIndex"`
	SourceStartValidatorIndex *uint64  `yaml:"sourceStartValidatorIndex" json:"sourceStartValidatorIndex"`
	SourceIndexCount          int      `yaml:"sourceIndexCount" json:"sourceIndexCount"`
	WithdrawAmount            uint64   `yaml:"withdrawAmount" json:"withdrawAmount"`
	WalletPrivkey             string   `yaml:"walletPrivkey" json:"walletPrivkey"`
	WithdrawalContract        string   `yaml:"withdrawalContract" json:"withdrawalContract"`
	TxAmount                  *big.Int `yaml:"txAmount" json:"txAmount"`
	TxFeeCap                  *big.Int `yaml:"txFeeCap" json:"txFeeCap"`
	TxTipCap                  *big.Int `yaml:"txTipCap" json:"txTipCap"`
	TxGasLimit                uint64   `yaml:"txGasLimit" json:"txGasLimit"`
	ClientPattern             string   `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern      string   `yaml:"excludeClientPattern" json:"excludeClientPattern"`
	AwaitReceipt              bool     `yaml:"awaitReceipt" json:"awaitReceipt"`
	FailOnReject              bool     `yaml:"failOnReject" json:"failOnReject"`
}

func DefaultConfig() Config {
	return Config{
		WithdrawalContract: "0x00A3ca265EBcb825B45F985A16CEFB49958cE017",
		TxAmount:           big.NewInt(500000000000000000), // 0.5 ETH
		TxFeeCap:           big.NewInt(100000000000),       // 100 Gwei
		TxTipCap:           big.NewInt(1000000000),         // 1 Gwei
		TxGasLimit:         200000,
	}
}

func (c *Config) Validate() error {
	if c.LimitTotal == 0 && c.LimitPerSlot == 0 {
		return errors.New("either limitTotal or limitPerSlot must be set")
	}

	if c.SourcePubkey == "" && c.SourceMnemonic == "" && c.SourceStartValidatorIndex == nil {
		return errors.New("either sourcePubkey or sourceMnemonic with sourceStartIndex or sourceStartValidatorIndex must be set")
	}

	return nil
}
