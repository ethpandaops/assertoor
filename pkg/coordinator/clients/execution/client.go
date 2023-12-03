package execution

import (
	"context"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethpandaops/minccino/pkg/coordinator/clients/execution/rpc"
	"github.com/sirupsen/logrus"
)

type ClientStatus uint8

var (
	ClientStatusOnline        ClientStatus = 1
	ClientStatusOffline       ClientStatus = 2
	ClientStatusSynchronizing ClientStatus = 3
)

type ClientConfig struct {
	Url     string
	Name    string
	Headers map[string]string
}

type Client struct {
	pool            *Pool
	clientIdx       uint16
	endpointConfig  *ClientConfig
	clientCtx       context.Context
	clientCtxCancel context.CancelFunc
	rpcClient       *rpc.ExecutionClient
	updateChan      chan *clientBlockNotification
	logger          *logrus.Entry
	isOnline        bool
	isSyncing       bool
	versionStr      string
	clientType      ClientType
	lastEvent       time.Time
	retryCounter    uint64
	lastError       error
	headMutex       sync.RWMutex
	headHash        common.Hash
	headNumber      uint64
	finalizedRoot   phase0.Root
	finalizedEpoch  phase0.Epoch
}

type clientBlockNotification struct {
	hash   common.Hash
	number uint64
}

func (pool *Pool) newPoolClient(clientIdx uint16, endpoint *ClientConfig) (*Client, error) {
	rpcClient, err := rpc.NewExecutionClient(endpoint.Name, endpoint.Url, endpoint.Headers)
	if err != nil {
		return nil, err
	}

	client := Client{
		pool:           pool,
		clientIdx:      clientIdx,
		endpointConfig: endpoint,
		rpcClient:      rpcClient,
		updateChan:     make(chan *clientBlockNotification, 10),
		logger:         logrus.WithField("client", endpoint.Name),
	}
	client.resetContext()
	go client.runClientLoop()
	return &client, nil
}

func (client *Client) resetContext() {
	if client.clientCtxCancel != nil {
		client.clientCtxCancel()
	}
	client.clientCtx, client.clientCtxCancel = context.WithCancel(context.Background())
}

func (client *Client) GetIndex() uint16 {
	return client.clientIdx
}

func (client *Client) GetName() string {
	return client.endpointConfig.Name
}

func (client *Client) GetVersion() string {
	return client.versionStr
}

func (client *Client) GetEndpointConfig() *ClientConfig {
	return client.endpointConfig
}

func (client *Client) GetLastHead() (uint64, common.Hash) {
	client.headMutex.RLock()
	defer client.headMutex.RUnlock()
	return client.headNumber, client.headHash
}

func (client *Client) GetLastError() error {
	return client.lastError
}

func (client *Client) GetLastEventTime() time.Time {
	return client.lastEvent
}

func (client *Client) GetRpcClient() *rpc.ExecutionClient {
	return client.rpcClient
}

func (client *Client) GetStatus() ClientStatus {
	if client.isSyncing {
		return ClientStatusSynchronizing
	} else if client.isOnline {
		return ClientStatusOnline
	} else {
		return ClientStatusOffline
	}
}

func (client *Client) NotifyNewBlock(hash common.Hash, number uint64) {
	if client.isOnline {
		client.updateChan <- &clientBlockNotification{
			hash:   hash,
			number: number,
		}
	}
}
