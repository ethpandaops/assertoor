package generateeoatransactions

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

	LegacyTxType       bool     `yaml:"legacyTxType" json:"legacyTxType"`
	FeeCap             *big.Int `yaml:"feeCap" json:"feeCap"`
	TipCap             *big.Int `yaml:"tipCap" json:"tipCap"`
	GasLimit           uint64   `yaml:"gasLimit" json:"gasLimit"`
	TargetAddress      string   `yaml:"targetAddress" json:"targetAddress"`
	RandomTarget       bool     `yaml:"randomTarget" json:"randomTarget"`
	ContractDeployment bool     `yaml:"contractDeployment" json:"contractDeployment"`
	CallData           string   `yaml:"callData" json:"callData"`
	RandomAmount       bool     `yaml:"randomAmount" json:"randomAmount"`
	Amount             *big.Int `yaml:"amount" json:"amount"`

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
