package generateeoatransactions

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerBlock int    `yaml:"limitPerBlock" json:"limitPerBlock" desc:"Maximum number of transactions to generate per block."`
	LimitTotal    int    `yaml:"limitTotal" json:"limitTotal" desc:"Total limit on the number of transactions to generate."`
	LimitPending  int    `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending transactions to allow before waiting."`
	PrivateKey    string `yaml:"privateKey" json:"privateKey" desc:"Private key of the wallet used to send transactions."`
	ChildWallets  uint64 `yaml:"childWallets" json:"childWallets" desc:"Number of child wallets to use for parallel transaction sending."`
	WalletSeed    string `yaml:"walletSeed" json:"walletSeed" desc:"Seed used to derive child wallet addresses deterministically."`

	RefillPendingLimit uint64   `yaml:"refillPendingLimit" json:"refillPendingLimit" desc:"Maximum pending refill transactions before waiting."`
	RefillFeeCap       *big.Int `yaml:"refillFeeCap" json:"refillFeeCap" desc:"Maximum fee cap (in wei) for refill transactions."`
	RefillTipCap       *big.Int `yaml:"refillTipCap" json:"refillTipCap" desc:"Maximum priority tip (in wei) for refill transactions."`
	RefillAmount       *big.Int `yaml:"refillAmount" json:"refillAmount" desc:"Amount (in wei) to transfer when refilling child wallets."`
	RefillMinBalance   *big.Int `yaml:"refillMinBalance" json:"refillMinBalance" desc:"Minimum balance (in wei) before triggering a refill."`

	LegacyTxType       bool     `yaml:"legacyTxType" json:"legacyTxType" desc:"If true, use legacy transaction type instead of EIP-1559."`
	FeeCap             *big.Int `yaml:"feeCap" json:"feeCap" desc:"Maximum fee cap (in wei) for generated transactions."`
	TipCap             *big.Int `yaml:"tipCap" json:"tipCap" desc:"Maximum priority tip (in wei) for generated transactions."`
	GasLimit           uint64   `yaml:"gasLimit" json:"gasLimit" desc:"Gas limit for generated transactions."`
	TargetAddress      string   `yaml:"targetAddress" json:"targetAddress" desc:"Target address to send transactions to."`
	RandomTarget       bool     `yaml:"randomTarget" json:"randomTarget" desc:"If true, send transactions to random addresses."`
	ContractDeployment bool     `yaml:"contractDeployment" json:"contractDeployment" desc:"If true, deploy contracts instead of simple transfers."`
	CallData           string   `yaml:"callData" json:"callData" desc:"Hex-encoded call data to include in transactions."`
	RandomAmount       bool     `yaml:"randomAmount" json:"randomAmount" desc:"If true, use random amounts for each transaction."`
	Amount             *big.Int `yaml:"amount" json:"amount" desc:"Amount (in wei) to send in each transaction."`

	AwaitReceipt  bool `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for transaction receipts before completing."`
	FailOnReject  bool `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if any transaction is rejected."`
	FailOnSuccess bool `yaml:"failOnSuccess" json:"failOnSuccess" desc:"Fail the task if any transaction succeeds (for negative testing)."`

	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting transactions."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
}

func DefaultConfig() Config {
	return Config{
		RefillPendingLimit: 200,
		RefillFeeCap:       big.NewInt(500000000000),        // 500 Gwei
		RefillTipCap:       big.NewInt(1000000000),          // 1 Gwei
		RefillAmount:       big.NewInt(1000000000000000000), // 1 ETH
		RefillMinBalance:   big.NewInt(500000000000000000),  // 0.5 ETH
		FeeCap:             big.NewInt(100000000000),        // 100 Gwei
		TipCap:             big.NewInt(1000000000),          // 1 Gwei
		GasLimit:           50000,
		Amount:             big.NewInt(0),
	}
}

func (c *Config) Validate() error {
	if c.LimitPerBlock == 0 && c.LimitTotal == 0 && c.LimitPending == 0 {
		return errors.New("either limitPerBlock or limitTotal or limitPending must be set")
	}

	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}
