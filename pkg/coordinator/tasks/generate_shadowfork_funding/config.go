package generateshadowforkfunding

import (
	"errors"
	"math/big"
)

type Config struct {
	ShadowForkVaultContract string `yaml:"shadowForkVaultContract" json:"shadowForkVaultContract"`

	PrivateKey string   `yaml:"privateKey" json:"privateKey"`
	MinBalance *big.Int `yaml:"minBalance" json:"minBalance"`

	TxFeeCap        *big.Int `yaml:"prefundFeeCap" json:"prefundFeeCap"`
	TxTipCap        *big.Int `yaml:"prefundTipCap" json:"prefundTipCap"`
	RequestAmount   *big.Int `yaml:"requestAmount" json:"requestAmount"`
	AwaitFeeFunding bool     `yaml:"awaitFeeFunding" json:"awaitFeeFunding"`

	TxHashResultVar string `yaml:"txHashResultVar" json:"txHashResultVar"`
}

func DefaultConfig() Config {
	return Config{
		ShadowForkVaultContract: "0x9620e3933dAAa49EBe3250b731291ac817E24372",

		MinBalance:    big.NewInt(1000000000000000000), // 1 ETH
		TxFeeCap:      big.NewInt(100000000000),        // 100 Gwei
		TxTipCap:      big.NewInt(1000000000),          // 1 Gwei
		RequestAmount: big.NewInt(1000000000000000000), // 1 ETH
	}
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}
