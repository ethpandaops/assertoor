package rpc

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type ExecutionClient struct {
	name             string
	endpoint         string
	headers          map[string]string
	rpcClient        *rpc.Client
	ethClient        *ethclient.Client
	concurrencyLimit int
	requestTimeout   time.Duration
	concurrencyChan  chan struct{}
}

// NewExecutionClient is used to create a new execution client
func NewExecutionClient(name, url string, headers map[string]string) (*ExecutionClient, error) {
	client := &ExecutionClient{
		name:             name,
		endpoint:         url,
		headers:          headers,
		concurrencyLimit: 50,
		requestTimeout:   30 * time.Second,
	}

	client.concurrencyChan = make(chan struct{}, client.concurrencyLimit)

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

func (ec *ExecutionClient) enforceConcurrencyLimit(ctx context.Context) func() {
	select {
	case <-ctx.Done():
		return func() {}
	case ec.concurrencyChan <- struct{}{}:
		return func() {
			<-ec.concurrencyChan
		}
	}
}

func (ec *ExecutionClient) GetEthClient() *ethclient.Client {
	return ec.ethClient
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

	return &ChainSpec{
		ChainID: chainID.String(),
	}, nil
}

func (ec *ExecutionClient) GetNodeSyncing(ctx context.Context) (*SyncStatus, error) {
	status, err := ec.ethClient.SyncProgress(ctx)
	if err != nil {
		return nil, err
	}

	if status == nil {
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
	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	block, err := ec.ethClient.BlockByNumber(reqCtx, nil)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (ec *ExecutionClient) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	block, err := ec.ethClient.BlockByHash(reqCtx, hash)
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (ec *ExecutionClient) GetNonceAt(ctx context.Context, wallet common.Address, blockNumber *big.Int) (uint64, error) {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return 0, fmt.Errorf("client busy")
	}

	defer closeFn()

	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	return ec.ethClient.NonceAt(reqCtx, wallet, blockNumber)
}

func (ec *ExecutionClient) GetBalanceAt(ctx context.Context, wallet common.Address, blockNumber *big.Int) (*big.Int, error) {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return nil, fmt.Errorf("client busy")
	}

	defer closeFn()

	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	return ec.ethClient.BalanceAt(reqCtx, wallet, blockNumber)
}

func (ec *ExecutionClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return nil, fmt.Errorf("client busy")
	}

	defer closeFn()

	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	return ec.ethClient.TransactionReceipt(reqCtx, txHash)
}

func (ec *ExecutionClient) GetBlockReceipts(ctx context.Context, blockHash common.Hash) ([]*types.Receipt, error) {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return nil, fmt.Errorf("client busy")
	}

	defer closeFn()

	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	return ec.ethClient.BlockReceipts(reqCtx, rpc.BlockNumberOrHash{
		BlockHash: &blockHash,
	})
}

func (ec *ExecutionClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return fmt.Errorf("client busy")
	}

	defer closeFn()

	reqCtx, reqCtxCancel := context.WithTimeout(ctx, ec.requestTimeout)
	defer reqCtxCancel()

	return ec.ethClient.SendTransaction(reqCtx, tx)
}

func (ec *ExecutionClient) GetEthCall(ctx context.Context, msg *ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	closeFn := ec.enforceConcurrencyLimit(ctx)
	if closeFn == nil {
		return nil, fmt.Errorf("client busy")
	}

	defer closeFn()

	return ec.ethClient.CallContract(ctx, *msg, blockNumber)
}
