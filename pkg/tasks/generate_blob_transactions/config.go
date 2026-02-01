package generateblobtransactions

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerBlock int    `yaml:"limitPerBlock" json:"limitPerBlock" desc:"Maximum number of blob transactions to generate per block."`
	LimitTotal    int    `yaml:"limitTotal" json:"limitTotal" desc:"Total limit on the number of blob transactions to generate."`
	LimitPending  int    `yaml:"limitPending" json:"limitPending" desc:"Maximum number of pending blob transactions to allow before waiting."`
	PrivateKey    string `yaml:"privateKey" json:"privateKey" desc:"Private key of the wallet used to send blob transactions."`
	ChildWallets  uint64 `yaml:"childWallets" json:"childWallets" desc:"Number of child wallets to use for parallel transaction sending."`
	WalletSeed    string `yaml:"walletSeed" json:"walletSeed" desc:"Seed used to derive child wallet addresses deterministically."`

	RefillPendingLimit uint64   `yaml:"refillPendingLimit" json:"refillPendingLimit" desc:"Maximum pending refill transactions before waiting."`
	RefillFeeCap       *big.Int `yaml:"refillFeeCap" json:"refillFeeCap" desc:"Maximum fee cap (in wei) for refill transactions."`
	RefillTipCap       *big.Int `yaml:"refillTipCap" json:"refillTipCap" desc:"Maximum priority tip (in wei) for refill transactions."`
	RefillAmount       *big.Int `yaml:"refillAmount" json:"refillAmount" desc:"Amount (in wei) to transfer when refilling child wallets."`
	RefillMinBalance   *big.Int `yaml:"refillMinBalance" json:"refillMinBalance" desc:"Minimum balance (in wei) before triggering a refill."`

	BlobSidecars  uint64   `yaml:"blobSidecars" json:"blobSidecars" desc:"Number of blob sidecars to include per transaction."`
	BlobFeeCap    *big.Int `yaml:"blobFeeCap" json:"blobFeeCap" desc:"Maximum blob fee cap (in wei) for blob transactions."`
	FeeCap        *big.Int `yaml:"feeCap" json:"feeCap" desc:"Maximum fee cap (in wei) for blob transactions."`
	TipCap        *big.Int `yaml:"tipCap" json:"tipCap" desc:"Maximum priority tip (in wei) for blob transactions."`
	GasLimit      uint64   `yaml:"gasLimit" json:"gasLimit" desc:"Gas limit for blob transactions."`
	TargetAddress string   `yaml:"targetAddress" json:"targetAddress" desc:"Target address to send blob transactions to."`
	RandomTarget  bool     `yaml:"randomTarget" json:"randomTarget" desc:"If true, send blob transactions to random addresses."`
	CallData      string   `yaml:"callData" json:"callData" desc:"Hex-encoded call data to include in blob transactions."`
	BlobData      string   `yaml:"blobData" json:"blobData" desc:"Hex-encoded blob data to use in blob sidecars."`
	RandomAmount  bool     `yaml:"randomAmount" json:"randomAmount" desc:"If true, use random amounts for each transaction."`
	Amount        *big.Int `yaml:"amount" json:"amount" desc:"Amount (in wei) to send in each blob transaction."`
	LegacyBlobTx  bool     `yaml:"legacyBlobTx" json:"legacyBlobTx" desc:"If true, use legacy blob transaction format."`

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
		BlobSidecars:       1,
		BlobFeeCap:         big.NewInt(10000000000),  // 10 Gwei
		FeeCap:             big.NewInt(100000000000), // 100 Gwei
		TipCap:             big.NewInt(2000000000),   // 2 Gwei
		GasLimit:           100000,
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
