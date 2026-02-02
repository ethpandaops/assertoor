package generatetransaction

import (
	"errors"
	"math/big"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey" desc:"Private key of the wallet used to send the transaction."`

	LegacyTxType       bool           `yaml:"legacyTxType" json:"legacyTxType" desc:"If true, use legacy transaction type instead of EIP-1559."`
	BlobTxType         bool           `yaml:"blobTxType" json:"blobTxType" desc:"If true, send a blob transaction (EIP-4844)."`
	SetCodeTxType      bool           `yaml:"setCodeTxType" json:"setCodeTxType" desc:"If true, send a set code transaction (EIP-7702)."`
	BlobFeeCap         *helper.BigInt `yaml:"blobFeeCap" json:"blobFeeCap" desc:"Maximum blob fee cap (in wei) for blob transactions."`
	FeeCap             *helper.BigInt `yaml:"feeCap" json:"feeCap" desc:"Maximum fee cap (in wei) for the transaction."`
	TipCap             *helper.BigInt `yaml:"tipCap" json:"tipCap" desc:"Maximum priority tip (in wei) for the transaction."`
	GasLimit           uint64         `yaml:"gasLimit" json:"gasLimit" desc:"Gas limit for the transaction."`
	TargetAddress      string         `yaml:"targetAddress" json:"targetAddress" desc:"Target address to send the transaction to."`
	RandomTarget       bool           `yaml:"randomTarget" json:"randomTarget" desc:"If true, send transaction to a random address."`
	ContractDeployment bool           `yaml:"contractDeployment" json:"contractDeployment" desc:"If true, deploy a contract instead of sending to an address."`
	CallData           string         `yaml:"callData" json:"callData" desc:"Hex-encoded call data to include in the transaction."`
	BlobData           string         `yaml:"blobData" json:"blobData" desc:"Hex-encoded blob data to use in blob sidecars."`
	BlobSidecars       uint64         `yaml:"blobSidecars" json:"blobSidecars" desc:"Number of blob sidecars to include in the transaction."`
	RandomAmount       bool           `yaml:"randomAmount" json:"randomAmount" desc:"If true, use a random amount for the transaction."`
	Amount             *helper.BigInt `yaml:"amount" json:"amount" desc:"Amount (in wei) to send in the transaction."`
	Nonce              *uint64        `yaml:"nonce" json:"nonce" desc:"Custom nonce to use for the transaction."`
	Authorizations     []struct {
		ChainID       uint64  `yaml:"chainId" json:"chainId" desc:"Chain ID for the authorization."`
		Nonce         *uint64 `yaml:"nonce" json:"nonce" desc:"Nonce for the authorization."`
		CodeAddress   string  `yaml:"codeAddress" json:"codeAddress" desc:"Code address for the authorization."`
		SignerPrivkey string  `yaml:"signerPrivkey" json:"signerPrivkey" desc:"Private key of the signer for the authorization."`
	} `yaml:"authorizations" json:"authorizations" desc:"List of authorizations for EIP-7702 set code transactions."`

	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for submitting the transaction."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`

	AwaitReceipt  bool `yaml:"awaitReceipt" json:"awaitReceipt" desc:"Wait for the transaction receipt before completing."`
	FailOnReject  bool `yaml:"failOnReject" json:"failOnReject" desc:"Fail the task if the transaction is rejected."`
	FailOnSuccess bool `yaml:"failOnSuccess" json:"failOnSuccess" desc:"Fail the task if the transaction succeeds (for negative testing)."`
	ExpectEvents  []struct {
		Topic0 string `yaml:"topic0" json:"topic0" desc:"Expected value for event topic 0 (event signature)."`
		Topic1 string `yaml:"topic1" json:"topic1" desc:"Expected value for event topic 1."`
		Topic2 string `yaml:"topic2" json:"topic2" desc:"Expected value for event topic 2."`
		Data   string `yaml:"data" json:"data" desc:"Expected event data."`
	} `yaml:"expectEvents" json:"expectEvents" desc:"List of events expected to be emitted by the transaction."`

	TransactionHashResultVar    string `yaml:"transactionHashResultVar" json:"transactionHashResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	TransactionReceiptResultVar string `yaml:"transactionReceiptResultVar" json:"transactionReceiptResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
	ContractAddressResultVar    string `yaml:"contractAddressResultVar" json:"contractAddressResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
}

func DefaultConfig() Config {
	return Config{
		FeeCap:       &helper.BigInt{Value: *big.NewInt(100000000000)}, // 100 Gwei
		TipCap:       &helper.BigInt{Value: *big.NewInt(1000000000)},   // 1 Gwei
		GasLimit:     50000,
		Amount:       &helper.BigInt{Value: *big.NewInt(0)},
		BlobSidecars: 1,
		AwaitReceipt: true,
	}
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" {
		return errors.New("privateKey must be set")
	}

	return nil
}
