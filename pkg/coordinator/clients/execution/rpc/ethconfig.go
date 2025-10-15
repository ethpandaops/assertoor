package rpc

import (
	"context"
	"encoding/json"
)

// EthConfigResponse represents the response from eth_config RPC call (EIP-7910)
type EthConfigResponse struct {
	Current *ForkConfig `json:"current"`
	Next    *ForkConfig `json:"next"`
	Last    *ForkConfig `json:"last"`
}

// ForkConfig represents a fork configuration
type ForkConfig struct {
	ActivationTime  int64                  `json:"activationTime"`
	BlobSchedule    map[string]interface{} `json:"blobSchedule,omitempty"`
	ChainID         string                 `json:"chainId"`
	ForkID          string                 `json:"forkId"`
	Precompiles     map[string]interface{} `json:"precompiles,omitempty"`
	SystemContracts map[string]interface{} `json:"systemContracts,omitempty"`
}

// GetEthConfig queries the eth_config RPC method (EIP-7910)
func (ec *ExecutionClient) GetEthConfig(ctx context.Context) (*EthConfigResponse, error) {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return nil, nil
	}

	defer closeFn()

	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	var result json.RawMessage
	err := ec.rpcClient.CallContext(reqCtx, &result, "eth_config")

	if err != nil {
		return nil, err
	}

	var config EthConfigResponse
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
