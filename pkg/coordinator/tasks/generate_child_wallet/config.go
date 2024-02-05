package generatechildwallet

import (
	"errors"
	"math/big"
)

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`
	WalletSeed string `yaml:"walletSeed" json:"walletSeed"`
	RandomSeed bool   `yaml:"randomSeed" json:"randomSeed"`

	PrefundFeeCap     *big.Int `yaml:"prefundFeeCap" json:"prefundFeeCap"`
	PrefundTipCap     *big.Int `yaml:"prefundTipCap" json:"prefundTipCap"`
	PrefundAmount     *big.Int `yaml:"prefundAmount" json:"prefundAmount"`
	PrefundMinBalance *big.Int `yaml:"prefundMinBalance" json:"prefundMinBalance"`

	WalletAddressResultVar    string `yaml:"walletAddressResultVar" json:"walletAddressResultVar"`
	WalletPrivateKeyResultVar string `yaml:"walletPrivateKeyResultVar" json:"walletPrivateKeyResultVar"`
}

func DefaultConfig() Config {
	return Config{
		PrefundFeeCap:     big.NewInt(500000000000),        // 500 Gwei
		PrefundTipCap:     big.NewInt(1000000000),          // 1 Gwei
		PrefundAmount:     big.NewInt(1000000000000000000), // 1 ETH
		PrefundMinBalance: big.NewInt(500000000000000000),  // 0.5 ETH
	}
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}
