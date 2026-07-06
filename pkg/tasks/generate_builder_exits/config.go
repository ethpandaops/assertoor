package generatebuilderexits

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerSlot         int      `yaml:"limitPerSlot" json:"limitPerSlot" require:"A.1" desc:"Maximum number of builder exits to generate per slot."`
	LimitTotal           int      `yaml:"limitTotal" json:"limitTotal" require:"A.2" desc:"Total limit on the number of builder exits to generate."`
	LimitPending         int      `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending builder exits to allow before waiting."`
	SourcePubkey         string   `yaml:"sourcePubkey" json:"sourcePubkey" require:"B.1" desc:"Public key of the builder to exit."`
	SourceMnemonic       string   `yaml:"sourceMnemonic" json:"sourceMnemonic" require:"B.2" desc:"Mnemonic phrase to derive builder keys for exits."`
	SourceStartIndex     int      `yaml:"sourceStartIndex" json:"sourceStartIndex" require:"B.2" desc:"Index within the mnemonic from which to start deriving builder keys."`
	SourceIndexCount     int      `yaml:"sourceIndexCount" json:"sourceIndexCount" desc:"Number of builders to generate exit requests for."`
	WalletPrivkey        string   `yaml:"walletPrivkey" json:"walletPrivkey" require:"C" desc:"Private key of the wallet used to send builder exit transactions. Must match the builder's execution address (its 0xB0 withdrawal credentials)."`
	BuilderExitContract  string   `yaml:"builderExitContract" json:"builderExitContract" desc:"Address of the builder exit system contract (EIP-8282)."`
	TxAmount             *big.Int `yaml:"txAmount" json:"txAmount" desc:"Amount of ETH (in wei) to send with the builder exit transaction to cover the request fee."`
	TxFeeCap             *big.Int `yaml:"txFeeCap" json:"txFeeCap" desc:"Maximum fee cap (in wei) for builder exit transactions."`
	TxTipCap             *big.Int `yaml:"txTipCap" json:"txTipCap" desc:"Maximum priority tip (in wei) for builder exit transactions."`
	TxGasLimit           uint64   `yaml:"txGasLimit" json:"txGasLimit" desc:"Gas limit for builder exit transactions."`
	ClientPattern        string   `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern string   `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitReceipt         bool     `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for transaction receipts before completing."`
	FailOnReject         bool     `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any transaction is rejected."`
}

func DefaultConfig() Config {
	return Config{
		BuilderExitContract: "0x000014574A74c805590AFF9499fc7A690f008282",
		TxAmount:            big.NewInt(100000000000000000), // 0.1 ETH
		TxFeeCap:            big.NewInt(100000000000),       // 100 Gwei
		TxTipCap:            big.NewInt(1000000000),         // 1 Gwei
		TxGasLimit:          1000000,
	}
}

func (c *Config) Validate() error {
	if c.LimitTotal == 0 && c.LimitPerSlot == 0 && c.SourceIndexCount == 0 {
		return errors.New("either limitTotal, limitPerSlot or sourceIndexCount must be set")
	}

	if c.SourcePubkey == "" && c.SourceMnemonic == "" {
		return errors.New("either sourcePubkey or sourceMnemonic with sourceStartIndex must be set")
	}

	if c.WalletPrivkey == "" {
		return errors.New("walletPrivkey must be set")
	}

	if c.BuilderExitContract == "" {
		return errors.New("builderExitContract must be set")
	}

	return nil
}
