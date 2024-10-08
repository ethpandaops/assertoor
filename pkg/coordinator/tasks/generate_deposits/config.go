package generatedeposits

import (
	"errors"
)

type Config struct {
	LimitPerSlot          int    `yaml:"limitPerSlot" json:"limitPerSlot"`
	LimitTotal            int    `yaml:"limitTotal" json:"limitTotal"`
	LimitPending          int    `yaml:"limitPending" json:"limitPending"`
	Mnemonic              string `yaml:"mnemonic" json:"mnemonic"`
	StartIndex            int    `yaml:"startIndex" json:"startIndex"`
	IndexCount            int    `yaml:"indexCount" json:"indexCount"`
	PublicKey             string `yaml:"publicKey" json:"publicKey"`
	WalletPrivkey         string `yaml:"walletPrivkey" json:"walletPrivkey"`
	DepositContract       string `yaml:"depositContract" json:"depositContract"`
	DepositAmount         uint64 `yaml:"depositAmount" json:"depositAmount"`
	DepositTxFeeCap       int64  `yaml:"depositTxFeeCap" json:"depositTxFeeCap"`
	DepositTxTipCap       int64  `yaml:"depositTxTipCap" json:"depositTxTipCap"`
	WithdrawalCredentials string `yaml:"withdrawalCredentials" json:"withdrawalCredentials"`
	TopUpDeposit          bool   `yaml:"topUpDeposit" json:"topUpDeposit"`
	ClientPattern         string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern  string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
	AwaitReceipt          bool   `yaml:"awaitReceipt" json:"awaitReceipt"`
	FailOnReject          bool   `yaml:"failOnReject" json:"failOnReject"`

	DepositTransactionsResultVar string `yaml:"depositTransactionsResultVar" json:"depositTransactionsResultVar"`
	DepositReceiptsResultVar     string `yaml:"depositReceiptsResultVar" json:"depositReceiptsResultVar"`
	ValidatorPubkeysResultVar    string `yaml:"validatorPubkeysResultVar" json:"validatorPubkeysResultVar"`
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
