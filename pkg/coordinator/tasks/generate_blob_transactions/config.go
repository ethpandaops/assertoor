package generateblobtransactions

import (
	"errors"
	"math/big"
)

type Config struct {
	LimitPerBlock int    `yaml:"limitPerBlock" json:"limitPerBlock"`
	LimitTotal    int    `yaml:"limitTotal" json:"limitTotal"`
	LimitPending  int    `yaml:"limitPending" json:"limitPending"`
	PrivateKey    string `yaml:"privateKey" json:"privateKey"`
	ChildWallets  uint64 `yaml:"childWallets" json:"childWallets"`
	WalletSeed    string `yaml:"walletSeed" json:"walletSeed"`

	RefillPendingLimit uint64   `yaml:"refillPendingLimit" json:"refillPendingLimit"`
	RefillFeeCap       *big.Int `yaml:"refillFeeCap" json:"refillFeeCap"`
	RefillTipCap       *big.Int `yaml:"refillTipCap" json:"refillTipCap"`
	RefillAmount       *big.Int `yaml:"refillAmount" json:"refillAmount"`
	RefillMinBalance   *big.Int `yaml:"refillMinBalance" json:"refillMinBalance"`

	BlobSidecars  uint64   `yaml:"blobSidecars" json:"blobSidecars"`
	BlobFeeCap    *big.Int `yaml:"blobFeeCap" json:"blobFeeCap"`
	FeeCap        *big.Int `yaml:"feeCap" json:"feeCap"`
	TipCap        *big.Int `yaml:"tipCap" json:"tipCap"`
	GasLimit      uint64   `yaml:"gasLimit" json:"gasLimit"`
	TargetAddress string   `yaml:"targetAddress" json:"targetAddress"`
	RandomTarget  bool     `yaml:"randomTarget" json:"randomTarget"`
	CallData      string   `yaml:"callData" json:"callData"`
	BlobData      string   `yaml:"blobData" json:"blobData"`
	RandomAmount  bool     `yaml:"randomAmount" json:"randomAmount"`
	Amount        *big.Int `yaml:"amount" json:"amount"`

	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
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
