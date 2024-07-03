package generateconsolidations

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerSlot              int      `yaml:"limitPerSlot" json:"limitPerSlot"`
	LimitTotal                int      `yaml:"limitTotal" json:"limitTotal"`
	LimitPending              int      `yaml:"limitPending" json:"limitPending"`
	SourceMnemonic            string   `yaml:"sourceMnemonic" json:"sourceMnemonic"`
	SourceStartIndex          int      `yaml:"sourceStartIndex" json:"sourceStartIndex"`
	SourceStartValidatorIndex *uint64  `yaml:"sourceStartValidatorIndex" json:"sourceStartValidatorIndex"`
	SourceIndexCount          int      `yaml:"sourceIndexCount" json:"sourceIndexCount"`
	TargetValidatorIndex      *uint64  `yaml:"targetValidatorIndex" json:"targetValidatorIndex"`
	ConsolidationEpoch        *uint64  `yaml:"consolidationEpoch" json:"consolidationEpoch"`
	WalletPrivkey             string   `yaml:"walletPrivkey" json:"walletPrivkey"`
	ConsolidationContract     string   `yaml:"consolidationContract" json:"consolidationContract"`
	TxAmount                  *big.Int `yaml:"txAmount" json:"txAmount"`
	TxFeeCap                  *big.Int `yaml:"txFeeCap" json:"txFeeCap"`
	TxTipCap                  *big.Int `yaml:"txTipCap" json:"txTipCap"`
	TxGasLimit                uint64   `yaml:"txGasLimit" json:"txGasLimit"`
	ClientPattern             string   `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern      string   `yaml:"excludeClientPattern" json:"excludeClientPattern"`
	AwaitReceipt              bool     `yaml:"awaitReceipt" json:"awaitReceipt"`
	FailOnReject              bool     `yaml:"failOnReject" json:"failOnReject"`

	ConsolidationTransactionsResultVar string `yaml:"consolidationTransactionsResultVar" json:"consolidationTransactionsResultVar"`
	ConsolidationReceiptsResultVar     string `yaml:"consolidationReceiptsResultVar" json:"consolidationReceiptsResultVar"`
}

func DefaultConfig() Config {
	return Config{
		ConsolidationContract: "0x00b42dbF2194e931E80326D950320f7d9Dbeac02",
		TxAmount:              big.NewInt(500000000000000000), // 0.5 ETH
		TxFeeCap:              big.NewInt(100000000000),       // 100 Gwei
		TxTipCap:              big.NewInt(1000000000),         // 1 Gwei
		TxGasLimit:            100000,
	}
}

func (c *Config) Validate() error {
	if c.LimitTotal == 0 && c.SourceIndexCount == 0 {
		return errors.New("either limitTotal or indexCount must be set")
	}

	if c.SourceMnemonic == "" && c.SourceStartValidatorIndex == nil {
		return errors.New("either sourceMnemonic with sourceStartIndex or sourceStartValidatorIndex must be set")
	}

	if c.TargetValidatorIndex == nil {
		return errors.New("targetValidatorIndex must be set")
	}

	return nil
}
