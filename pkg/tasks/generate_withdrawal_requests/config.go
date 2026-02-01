package generatewithdrawalrequests

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerSlot              int      `yaml:"limitPerSlot" json:"limitPerSlot" desc:"Maximum number of withdrawal requests to generate per slot."`
	LimitTotal                int      `yaml:"limitTotal" json:"limitTotal" desc:"Total limit on the number of withdrawal requests to generate."`
	LimitPending              int      `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending withdrawal requests to allow before waiting."`
	SourcePubkey              string   `yaml:"sourcePubkey" json:"sourcePubkey" desc:"Public key of the validator to withdraw from."`
	SourceMnemonic            string   `yaml:"sourceMnemonic" json:"sourceMnemonic" desc:"Mnemonic phrase to derive validator keys for withdrawal."`
	SourceStartIndex          int      `yaml:"sourceStartIndex" json:"sourceStartIndex" desc:"Index within the mnemonic from which to start deriving keys."`
	SourceStartValidatorIndex *uint64  `yaml:"sourceStartValidatorIndex" json:"sourceStartValidatorIndex" desc:"Starting validator index for withdrawal requests."`
	SourceIndexCount          int      `yaml:"sourceIndexCount" json:"sourceIndexCount" desc:"Number of validators to generate withdrawal requests for."`
	WithdrawAmount            uint64   `yaml:"withdrawAmount" json:"withdrawAmount" desc:"Amount of ETH to withdraw from each validator."`
	WalletPrivkey             string   `yaml:"walletPrivkey" json:"walletPrivkey" desc:"Private key of the wallet used to send withdrawal request transactions."`
	WithdrawalContract        string   `yaml:"withdrawalContract" json:"withdrawalContract" desc:"Address of the withdrawal request contract."`
	TxAmount                  *big.Int `yaml:"txAmount" json:"txAmount" desc:"Amount of ETH to send with the withdrawal request transaction."`
	TxFeeCap                  *big.Int `yaml:"txFeeCap" json:"txFeeCap" desc:"Maximum fee cap (in wei) for withdrawal request transactions."`
	TxTipCap                  *big.Int `yaml:"txTipCap" json:"txTipCap" desc:"Maximum priority tip (in wei) for withdrawal request transactions."`
	TxGasLimit                uint64   `yaml:"txGasLimit" json:"txGasLimit" desc:"Gas limit for withdrawal request transactions."`
	ClientPattern             string   `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern      string   `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitReceipt              bool     `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for transaction receipts before completing."`
	FailOnReject              bool     `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any transaction is rejected."`
}

func DefaultConfig() Config {
	return Config{
		WithdrawalContract: "0x00000961Ef480Eb55e80D19ad83579A64c007002",
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
