package generateconsolidations

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerSlot              int      `yaml:"limitPerSlot" json:"limitPerSlot" desc:"Maximum number of consolidation requests to generate per slot."`
	LimitTotal                int      `yaml:"limitTotal" json:"limitTotal" desc:"Total limit on the number of consolidation requests to generate."`
	LimitPending              int      `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending consolidation requests to allow before waiting."`
	SourceMnemonic            string   `yaml:"sourceMnemonic" json:"sourceMnemonic" desc:"Mnemonic phrase to derive source validator keys."`
	SourceStartIndex          int      `yaml:"sourceStartIndex" json:"sourceStartIndex" desc:"Index within the mnemonic from which to start deriving source keys."`
	SourceStartValidatorIndex *uint64  `yaml:"sourceStartValidatorIndex" json:"sourceStartValidatorIndex" desc:"Starting validator index for source validators."`
	SourceIndexCount          int      `yaml:"sourceIndexCount" json:"sourceIndexCount" desc:"Number of source validators to consolidate."`
	TargetPublicKey           string   `yaml:"targetPublicKey" json:"targetPublicKey" desc:"Public key of the target validator to consolidate into."`
	TargetValidatorIndex      *uint64  `yaml:"targetValidatorIndex" json:"targetValidatorIndex" desc:"Validator index of the target validator to consolidate into."`
	ConsolidationEpoch        *uint64  `yaml:"consolidationEpoch" json:"consolidationEpoch" desc:"Epoch at which consolidation should occur."`
	WalletPrivkey             string   `yaml:"walletPrivkey" json:"walletPrivkey" desc:"Private key of the wallet used to send consolidation request transactions."`
	ConsolidationContract     string   `yaml:"consolidationContract" json:"consolidationContract" desc:"Address of the consolidation request contract."`
	TxAmount                  *big.Int `yaml:"txAmount" json:"txAmount" desc:"Amount of ETH to send with the consolidation request transaction."`
	TxFeeCap                  *big.Int `yaml:"txFeeCap" json:"txFeeCap" desc:"Maximum fee cap (in wei) for consolidation request transactions."`
	TxTipCap                  *big.Int `yaml:"txTipCap" json:"txTipCap" desc:"Maximum priority tip (in wei) for consolidation request transactions."`
	TxGasLimit                uint64   `yaml:"txGasLimit" json:"txGasLimit" desc:"Gas limit for consolidation request transactions."`
	ClientPattern             string   `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern      string   `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitReceipt              bool     `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for transaction receipts before completing."`
	FailOnReject              bool     `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any transaction is rejected."`
}

func DefaultConfig() Config {
	return Config{
		ConsolidationContract: "0x0000BBdDc7CE488642fb579F8B00f3a590007251",
		TxAmount:              big.NewInt(500000000000000000), // 0.5 ETH
		TxFeeCap:              big.NewInt(100000000000),       // 100 Gwei
		TxTipCap:              big.NewInt(1000000000),         // 1 Gwei
		TxGasLimit:            200000,
	}
}

func (c *Config) Validate() error {
	if c.LimitTotal == 0 && c.SourceIndexCount == 0 {
		return errors.New("either limitTotal or indexCount must be set")
	}

	if c.SourceMnemonic == "" && c.SourceStartValidatorIndex == nil {
		return errors.New("either sourceMnemonic with sourceStartIndex or sourceStartValidatorIndex must be set")
	}

	if c.TargetValidatorIndex == nil && c.TargetPublicKey == "" {
		return errors.New("either targetValidatorIndex or targetPublicKey must be set")
	}

	return nil
}
