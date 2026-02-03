package generatechildwallet

import (
	"errors"
	"math/big"
)

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey" require:"A" desc:"Private key of the parent wallet used to fund the child wallet."`
	WalletSeed string `yaml:"walletSeed" json:"walletSeed" desc:"Seed used to derive the child wallet address deterministically."`
	RandomSeed bool   `yaml:"randomSeed" json:"randomSeed" desc:"If true, generate a random seed for the child wallet."`

	PrefundFeeCap     *big.Int `yaml:"prefundFeeCap" json:"prefundFeeCap" desc:"Maximum fee cap (in wei) for the prefunding transaction."`
	PrefundTipCap     *big.Int `yaml:"prefundTipCap" json:"prefundTipCap" desc:"Maximum priority tip (in wei) for the prefunding transaction."`
	PrefundAmount     *big.Int `yaml:"prefundAmount" json:"prefundAmount" desc:"Amount (in wei) to transfer to the child wallet."`
	PrefundMinBalance *big.Int `yaml:"prefundMinBalance" json:"prefundMinBalance" desc:"Minimum balance (in wei) before triggering a prefund."`

	KeepFunding bool `yaml:"keepFunding" json:"keepFunding" desc:"If true, keep the wallet pool funding loop running after initial distribution. Default is false."`

	WalletAddressResultVar    string `yaml:"walletAddressResultVar" json:"walletAddressResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	WalletPrivateKeyResultVar string `yaml:"walletPrivateKeyResultVar" json:"walletPrivateKeyResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
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
