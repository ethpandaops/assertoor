package generatebatchdeposits

import (
	"errors"
)

type Config struct {
	LimitPerSlot          int    `yaml:"limitPerSlot" json:"limitPerSlot" require:"A.1" desc:"Maximum number of deposits to generate per slot. Counted in deposits, not batches."`
	LimitTotal            int    `yaml:"limitTotal" json:"limitTotal" require:"A.2" desc:"Total limit on the number of deposits to generate."`
	LimitPendingBatches   int    `yaml:"limitPendingBatches" json:"limitPendingBatches" desc:"Maximum number of pending batch transactions to allow before waiting."`
	Mnemonic              string `yaml:"mnemonic" json:"mnemonic" require:"B" desc:"Mnemonic phrase used to generate validator keys."`
	StartIndex            int    `yaml:"startIndex" json:"startIndex" desc:"Index within the mnemonic from which to start generating validator keys."`
	IndexCount            int    `yaml:"indexCount" json:"indexCount" require:"A.3" desc:"Number of validator keys to generate from the mnemonic."`
	WalletPrivkey         string `yaml:"walletPrivkey" json:"walletPrivkey" require:"C" desc:"Private key of the wallet used to fund deposit transactions and (if needed) deploy the batch contract."`
	DepositContract       string `yaml:"depositContract" json:"depositContract" require:"D" desc:"Address of the beacon chain deposit contract on the execution layer."`
	BatchContract         string `yaml:"batchContract" json:"batchContract" desc:"Address of an already-deployed BatchDeposit forwarder contract. If empty, a fresh contract is deployed at task start."`
	BatchSize             int    `yaml:"batchSize" json:"batchSize" desc:"Number of deposits to bundle into a single transaction. Default 100."`
	BatchTxGasLimit       uint64 `yaml:"batchTxGasLimit" json:"batchTxGasLimit" desc:"Gas limit for each batched deposit transaction. Default 12,000,000."`
	DepositAmount         uint64 `yaml:"depositAmount" json:"depositAmount" desc:"Amount of ETH to deposit per validator. Default 32 ETH."`
	DepositTxFeeCap       int64  `yaml:"depositTxFeeCap" json:"depositTxFeeCap" desc:"Maximum fee cap (in wei) for batch transactions."`
	DepositTxTipCap       int64  `yaml:"depositTxTipCap" json:"depositTxTipCap" desc:"Maximum priority tip (in wei) for batch transactions."`
	WithdrawalCredentials string `yaml:"withdrawalCredentials" json:"withdrawalCredentials" require:"E" desc:"32-byte withdrawal credentials shared by all deposits in every batch. For 0x03 builder credentials use '0x03' + 11 zero bytes + 20-byte address."`
	ClientPattern         string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern  string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	AwaitReceipt          bool   `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for batch transaction receipts on the execution layer before completing."`
	FailOnReject          bool   `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any batch transaction is rejected."`
	AwaitInclusion        bool   `yaml:"awaitInclusion" json:"awaitInclusion" desc:"Wait for all generated deposits to be included in beacon blocks before completing."`
}

func DefaultConfig() Config {
	return Config{
		// Total tx fee = DepositTxFeeCap * BatchTxGasLimit.
		// Keep under 1 ETH to satisfy geth's default --rpc.txfeecap=1eth.
		// With 12M gas, max safe fee cap is ~83 gwei. We use 50 gwei for headroom.
		DepositTxFeeCap: 50000000000, // 50 gwei
		DepositTxTipCap: 1000000000,  // 1 gwei
		DepositAmount:   32,          // 32 ETH
		BatchSize:       100,
		BatchTxGasLimit: 12000000,
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

	if c.WithdrawalCredentials == "" {
		return errors.New("withdrawalCredentials must be set")
	}

	if c.DepositAmount == 0 {
		return errors.New("depositAmount must be > 0")
	}

	if c.BatchSize <= 0 {
		return errors.New("batchSize must be > 0")
	}

	if c.BatchTxGasLimit == 0 {
		return errors.New("batchTxGasLimit must be > 0")
	}

	return nil
}
