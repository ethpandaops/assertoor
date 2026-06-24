package generatebuilderdeposits

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerSlot           int      `yaml:"limitPerSlot" json:"limitPerSlot" require:"A.1" desc:"Maximum number of builder deposits to generate per slot."`
	LimitTotal             int      `yaml:"limitTotal" json:"limitTotal" require:"A.2" desc:"Total limit on the number of builder deposits to generate."`
	LimitPending           int      `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending builder deposits to allow before waiting."`
	Mnemonic               string   `yaml:"mnemonic" json:"mnemonic" require:"B.1" desc:"Mnemonic phrase used to generate builder BLS keys."`
	StartIndex             int      `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start generating builder keys."`
	IndexCount             int      `yaml:"indexCount" json:"indexCount" require:"A.3" desc:"Number of builder keys to generate from the mnemonic."`
	PublicKey              string   `yaml:"publicKey" json:"publicKey" require:"B.2" desc:"Public key of an existing builder for top-up deposits (requires topUpDeposit)."`
	WalletPrivkey          string   `yaml:"walletPrivkey" json:"walletPrivkey" require:"C" desc:"Private key of the wallet used to fund builder deposit transactions."`
	BuilderDepositContract string   `yaml:"builderDepositContract" json:"builderDepositContract" desc:"Address of the builder deposit system contract (EIP-8282)."`
	DepositAmount          uint64   `yaml:"depositAmount" json:"depositAmount" desc:"Amount of ETH to deposit per builder (must be >= 1 ETH)."`
	WithdrawalCredentials  string   `yaml:"withdrawalCredentials" json:"withdrawalCredentials" desc:"Custom withdrawal credentials to use (must be 0x03-prefixed). If empty, derived from builderAddress or the funding wallet address."`
	BuilderAddress         string   `yaml:"builderAddress" json:"builderAddress" desc:"Execution address used to build 0x03 withdrawal credentials when withdrawalCredentials is not set. This address is the only one allowed to exit the builder."`
	TopUpDeposit           bool     `yaml:"topUpDeposit" json:"topUpDeposit" desc:"If true, add to an existing builder balance instead of registering a new builder. Withdrawal credentials and signature are ignored by the consensus layer for top-ups."`
	TxFeeCap               *big.Int `yaml:"txFeeCap" json:"txFeeCap" desc:"Maximum fee cap (in wei) for builder deposit transactions."`
	TxTipCap               *big.Int `yaml:"txTipCap" json:"txTipCap" desc:"Maximum priority tip (in wei) for builder deposit transactions."`
	TxGasLimit             uint64   `yaml:"txGasLimit" json:"txGasLimit" desc:"Gas limit for builder deposit transactions."`
	TxFeeBuffer            *big.Int `yaml:"txFeeBuffer" json:"txFeeBuffer" desc:"Extra value (in wei) sent on top of the deposit amount to cover the request fee."`
	ClientPattern          string   `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern   string   `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitReceipt           bool     `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for transaction receipts on the execution layer before completing."`
	FailOnReject           bool     `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any builder deposit transaction is rejected."`
	AwaitInclusion         bool     `yaml:"awaitInclusion" json:"awaitInclusion" desc:"Wait for builder deposits to be included in beacon blocks before completing."`
	InvalidSigPercent      int      `yaml:"invalidSigPercent" json:"invalidSigPercent" desc:"Random percentage (0-100) of deposits to generate with corrupted signatures."`
}

func DefaultConfig() Config {
	return Config{
		BuilderDepositContract: "0x0000884d2AA32eAa155F59A2f24eFa73D9008282",
		DepositAmount:          1,                        // 1 ETH (BUILDER_MIN_DEPOSIT)
		TxFeeCap:               big.NewInt(100000000000), // 100 gwei
		TxTipCap:               big.NewInt(1000000000),   // 1 gwei
		TxGasLimit:             2000000,
		TxFeeBuffer:            big.NewInt(100000000000000000), // 0.1 ETH
	}
}

func (c *Config) Validate() error {
	if c.LimitPerSlot == 0 && c.LimitTotal == 0 && c.IndexCount == 0 {
		return errors.New("either limitPerSlot or limitTotal or indexCount must be set")
	}

	switch {
	case c.Mnemonic == "" && c.PublicKey == "":
		return errors.New("mnemonic or publicKey must be set")
	case c.Mnemonic != "" && c.PublicKey != "":
		return errors.New("only one of mnemonic or publicKey must be set")
	case c.PublicKey != "" && !c.TopUpDeposit:
		return errors.New("publicKey can only be used with topUpDeposit")
	}

	if c.WalletPrivkey == "" {
		return errors.New("walletPrivkey must be set")
	}

	if c.BuilderDepositContract == "" {
		return errors.New("builderDepositContract must be set")
	}

	if c.DepositAmount == 0 {
		return errors.New("depositAmount must be > 0")
	}

	if c.InvalidSigPercent < 0 || c.InvalidSigPercent > 100 {
		return errors.New("invalidSigPercent must be in range 0-100")
	}

	return nil
}
