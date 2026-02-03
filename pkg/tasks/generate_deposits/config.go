package generatedeposits

import (
	"errors"
)

type Config struct {
	LimitPerSlot          int    `yaml:"limitPerSlot" json:"limitPerSlot" require:"A.1" desc:"Maximum number of deposit operations to generate per slot."`
	LimitTotal            int    `yaml:"limitTotal" json:"limitTotal" require:"A.2" desc:"Total limit on the number of deposit operations to generate."`
	LimitPending          int    `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending deposits to allow before waiting."`
	Mnemonic              string `yaml:"mnemonic" json:"mnemonic" require:"B.1" desc:"Mnemonic phrase used to generate validator keys."`
	StartIndex            int    `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start generating validator keys."`
	IndexCount            int    `yaml:"indexCount" json:"indexCount" require:"A.3" desc:"Number of validator keys to generate from the mnemonic."`
	PublicKey             string `yaml:"publicKey" json:"publicKey" require:"B.2" desc:"Public key of an existing validator for top-up deposits (requires topUpDeposit)."`
	WalletPrivkey         string `yaml:"walletPrivkey" json:"walletPrivkey" require:"C" desc:"Private key of the wallet used to fund deposit transactions."`
	DepositContract       string `yaml:"depositContract" json:"depositContract" require:"D" desc:"Address of the deposit contract on the execution layer."`
	DepositAmount         uint64 `yaml:"depositAmount" json:"depositAmount" desc:"Amount of ETH to deposit per validator."`
	DepositTxFeeCap       int64  `yaml:"depositTxFeeCap" json:"depositTxFeeCap" desc:"Maximum fee cap (in wei) for deposit transactions."`
	DepositTxTipCap       int64  `yaml:"depositTxTipCap" json:"depositTxTipCap" desc:"Maximum priority tip (in wei) for deposit transactions."`
	WithdrawalCredentials string `yaml:"withdrawalCredentials" json:"withdrawalCredentials" desc:"Custom withdrawal credentials to use for deposits."`
	TopUpDeposit          bool   `yaml:"topUpDeposit" json:"topUpDeposit" desc:"If true, add to existing validator balance instead of creating new validators."`
	ClientPattern         string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern  string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitReceipt          bool   `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for transaction receipts on the execution layer before completing."`
	FailOnReject          bool   `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any deposit transaction is rejected."`
	AwaitInclusion        bool   `yaml:"awaitInclusion" json:"awaitInclusion" desc:"Wait for deposits to be included in beacon blocks before completing."`

	DepositTransactionsResultVar string `yaml:"depositTransactionsResultVar" json:"depositTransactionsResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	DepositReceiptsResultVar     string `yaml:"depositReceiptsResultVar" json:"depositReceiptsResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	ValidatorPubkeysResultVar    string `yaml:"validatorPubkeysResultVar" json:"validatorPubkeysResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
}

func DefaultConfig() Config {
	return Config{
		DepositTxFeeCap: 100000000000, // 100 gwei
		DepositTxTipCap: 1000000000,   // 1 gwei
		DepositAmount:   32,           // 32 ETH
	}
}

func (c *Config) Validate() error {
	if c.LimitPerSlot == 0 && c.LimitTotal == 0 && c.IndexCount == 0 {
		return errors.New("either limitPerSlot or limitTotal or indexCount must be set")
	}

	switch {
	case c.Mnemonic == "" && c.PublicKey == "":
		return errors.New("mnemonic or publickey must be set")
	case c.Mnemonic != "" && c.PublicKey != "":
		return errors.New("only one of mnemonic or publickey must be set")
	case c.PublicKey != "" && !c.TopUpDeposit:
		return errors.New("publicKey can only be used with topUpDeposit")
	}

	if c.WalletPrivkey == "" {
		return errors.New("walletPrivkey must be set")
	}

	if c.DepositContract == "" {
		return errors.New("depositContract must be set")
	}

	if c.DepositAmount == 0 {
		return errors.New("depositAmount must be > 0")
	}

	return nil
}
