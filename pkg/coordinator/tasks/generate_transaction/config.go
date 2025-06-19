package generatetransaction

import (
	"errors"
	"math/big"

	"github.com/erigontech/assertoor/pkg/coordinator/helper"
)

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	LegacyTxType       bool           `yaml:"legacyTxType" json:"legacyTxType"`
	BlobTxType         bool           `yaml:"blobTxType" json:"blobTxType"`
	SetCodeTxType      bool           `yaml:"setCodeTxType" json:"setCodeTxType"`
	BlobFeeCap         *helper.BigInt `yaml:"blobFeeCap" json:"blobFeeCap"`
	FeeCap             *helper.BigInt `yaml:"feeCap" json:"feeCap"`
	TipCap             *helper.BigInt `yaml:"tipCap" json:"tipCap"`
	GasLimit           uint64         `yaml:"gasLimit" json:"gasLimit"`
	TargetAddress      string         `yaml:"targetAddress" json:"targetAddress"`
	RandomTarget       bool           `yaml:"randomTarget" json:"randomTarget"`
	ContractDeployment bool           `yaml:"contractDeployment" json:"contractDeployment"`
	CallData           string         `yaml:"callData" json:"callData"`
	BlobData           string         `yaml:"blobData" json:"blobData"`
	RandomAmount       bool           `yaml:"randomAmount" json:"randomAmount"`
	Amount             *helper.BigInt `yaml:"amount" json:"amount"`
	Nonce              *uint64        `yaml:"nonce" json:"nonce"`
	Authorizations     []struct {
		ChainID       uint64  `yaml:"chainId" json:"chainId"`
		Nonce         *uint64 `yaml:"nonce" json:"nonce"`
		CodeAddress   string  `yaml:"codeAddress" json:"codeAddress"`
		SignerPrivkey string  `yaml:"signerPrivkey" json:"signerPrivkey"`
	} `yaml:"authorizations" json:"authorizations"`

	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`

	AwaitReceipt  bool `yaml:"awaitReceipt" json:"awaitReceipt"`
	FailOnReject  bool `yaml:"failOnReject" json:"failOnReject"`
	FailOnSuccess bool `yaml:"failOnSuccess" json:"failOnSuccess"`
	ExpectEvents  []struct {
		Topic0 string `yaml:"topic0" json:"topic0"`
		Topic1 string `yaml:"topic1" json:"topic1"`
		Topic2 string `yaml:"topic2" json:"topic2"`
		Data   string `yaml:"data" json:"data"`
	} `yaml:"expectEvents" json:"expectEvents"`

	TransactionHashResultVar    string `yaml:"transactionHashResultVar" json:"transactionHashResultVar"`
	TransactionReceiptResultVar string `yaml:"transactionReceiptResultVar" json:"transactionReceiptResultVar"`
	ContractAddressResultVar    string `yaml:"contractAddressResultVar" json:"contractAddressResultVar"`
}

func DefaultConfig() Config {
	return Config{
		FeeCap:       &helper.BigInt{Value: *big.NewInt(100000000000)}, // 100 Gwei
		TipCap:       &helper.BigInt{Value: *big.NewInt(1000000000)},   // 1 Gwei
		GasLimit:     50000,
		Amount:       &helper.BigInt{Value: *big.NewInt(0)},
		AwaitReceipt: true,
	}
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}
