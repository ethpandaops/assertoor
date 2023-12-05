package rpc

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type ExecutionClient struct {
	name      string
	endpoint  string
	headers   map[string]string
	rpcClient *rpc.Client
	ethClient *ethclient.Client
}

// NewExecutionClient is used to create a new execution client
func NewExecutionClient(name, url string, headers map[string]string) (*ExecutionClient, error) {
	client := &ExecutionClient{
		name:     name,
		endpoint: url,
		headers:  headers,
	}

	return client, nil
}

func (ec *ExecutionClient) Initialize(ctx context.Context) error {
	if ec.ethClient != nil {
		return nil
	}

	rpcClient, err := rpc.DialContext(ctx, ec.endpoint)
	if err != nil {
		return err
	}

	for hKey, hVal := range ec.headers {
		rpcClient.SetHeader(hKey, hVal)
	}

	ec.rpcClient = rpcClient
	ec.ethClient = ethclient.NewClient(rpcClient)

	return nil
}

func (ec *ExecutionClient) GetClientVersion(ctx context.Context) (string, error) {
	var result string
	err := ec.rpcClient.CallContext(ctx, &result, "web3_clientVersion")

	return result, err
}

func (ec *ExecutionClient) GetChainSpec(ctx context.Context) (*ChainSpec, error) {
	chainID, err := ec.ethClient.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	networkID, err := ec.ethClient.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	return &ChainSpec{
		ChainID:   chainID.String(),
		NetworkID: networkID.String(),
	}, nil
}

func (ec *ExecutionClient) GetNodeSyncing(ctx context.Context) (*SyncStatus, error) {
	status, err := ec.ethClient.SyncProgress(ctx)
	if err != nil {
		return nil, err
	}

	if status == nil && err == nil {
		// Not syncing
		ss := &SyncStatus{}
		ss.IsSyncing = false

		return ss, nil
	}

	return &SyncStatus{
		IsSyncing:     true,
		CurrentBlock:  status.CurrentBlock,
		HighestBlock:  status.HighestBlock,
		StartingBlock: status.StartingBlock,
	}, nil
}

func (ec *ExecutionClient) GetLatestBlock(ctx context.Context) (*types.Block, error) {
	block, err := ec.ethClient.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (ec *ExecutionClient) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := ec.ethClient.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}
