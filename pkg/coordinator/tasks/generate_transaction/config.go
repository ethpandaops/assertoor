package generatetransaction

import (
	"errors"
	"math/big"
)

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

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

	ClientPattern string `yaml:"clientPattern" json:"clientPattern"`

	AwaitReceipt  bool `yaml:"awaitReceipt" json:"awaitReceipt"`
	FailOnReject  bool `yaml:"failOnReject" json:"failOnReject"`
	FailOnSuccess bool `yaml:"failOnSuccess" json:"failOnSuccess"`
	ExpectEvents  []struct {
		Topic0 string `yaml:"topic0" json:"topic0"`
		Topic1 string `yaml:"topic1" json:"topic1"`
		Topic2 string `yaml:"topic2" json:"topic2"`
		Data   string `yaml:"data" json:"data"`
	} `yaml:"expectEvents" json:"expectEvents"`

	TransactionHashResultVar string `yaml:"transactionHashResultVar" json:"transactionHashResultVar"`
	ContractAddressResultVar string `yaml:"contractAddressResultVar" json:"contractAddressResultVar"`
}

func DefaultConfig() Config {
	return Config{
		FeeCap:       big.NewInt(100000000000), // 100 Gwei
		TipCap:       big.NewInt(1000000000),   // 1 Gwei
		GasLimit:     50000,
		Amount:       big.NewInt(0),
		AwaitReceipt: true,
	}
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}
