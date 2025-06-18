package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type PoolTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	GasFeeCap        *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	MaxFeePerBlobGas *hexutil.Big    `json:"maxFeePerBlobGas,omitempty"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	Type             hexutil.Uint64  `json:"type"`
	ChainID          *hexutil.Big    `json:"chainId,omitempty"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}
