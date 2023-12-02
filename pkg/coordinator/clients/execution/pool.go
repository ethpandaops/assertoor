package execution

import (
	"fmt"
	"sync"
)

type SchedulerMode uint8

var (
	RoundRobinScheduler SchedulerMode = 1
)

type PoolConfig struct {
	FollowDistance uint64 `yaml:"followDistance" envconfig:"EXECUTION_POOL_FOLLOW_DISTANCE"`
	ForkDistance   uint64 `yaml:"forkDistance" envconfig:"EXECUTION_POOL_FORK_DISTANCE"`
	SchedulerMode  string `yaml:"schedulerMode" envconfig:"EXECUTION_POOL_SCHEDULER_MODE"`
}

type Pool struct {
	config         *PoolConfig
	clientCounter  uint16
	clients        []*Client
	blockCache     *BlockCache
	forkCacheMutex sync.Mutex
	forkCache      map[int64][]*HeadFork

	schedulerMode  SchedulerMode
	schedulerMutex sync.Mutex
	rrLastIndexes  map[ClientType]uint16
}

func NewPool(config *PoolConfig) (*Pool, error) {
	pool := Pool{
		clients:       make([]*Client, 0),
		rrLastIndexes: map[ClientType]uint16{},
	}
	var err error

	switch config.SchedulerMode {
	case "", "rr", "roundrobin":
		pool.schedulerMode = RoundRobinScheduler
	default:
		return nil, fmt.Errorf("unknown pool schedulerMode: %v", config.SchedulerMode)
	}

	pool.blockCache, err = NewBlockCache(config.FollowDistance)
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (pool *Pool) GetBlockCache() *BlockCache {
	return pool.blockCache
}

func (pool *Pool) AddEndpoint(endpoint *ClientConfig) (*Client, error) {
	clientIdx := pool.clientCounter
	pool.clientCounter++
	client, err := pool.newPoolClient(clientIdx, endpoint)
	if err != nil {
		return nil, err
	}
	pool.clients = append(pool.clients, client)
	return client, nil
}

func (pool *Pool) GetAllEndpoints() []*Client {
	return pool.clients
}

func (pool *Pool) GetReadyEndpoint(clientType ClientType) *Client {
	canonicalFork := pool.GetCanonicalFork(-1)
	if canonicalFork == nil {
		return nil
	}

	readyClients := canonicalFork.ReadyClients
	if len(readyClients) == 0 {
		return nil
	}
	selectedClient := pool.runClientScheduler(readyClients, clientType)

	return selectedClient
}

func (pool *Pool) IsClientReady(client *Client) bool {
	if client == nil {
		return false
	}

	canonicalFork := pool.GetCanonicalFork(-1)
	if canonicalFork == nil {
		return false
	}

	readyClients := canonicalFork.ReadyClients
	for _, readyClient := range readyClients {
		if readyClient == client {
			return true
		}
	}
	return false
}

func (pool *Pool) runClientScheduler(readyClients []*Client, clientType ClientType) *Client {
	pool.schedulerMutex.Lock()
	defer pool.schedulerMutex.Unlock()

	switch pool.schedulerMode {
	case RoundRobinScheduler:
		var firstReadyClient *Client
		for _, client := range readyClients {
			if clientType != UnspecifiedClient && clientType != client.clientType {
				continue
			}
			if firstReadyClient == nil {
				firstReadyClient = client
			}
			if client.clientIdx > pool.rrLastIndexes[clientType] {
				pool.rrLastIndexes[clientType] = client.clientIdx
				return client
			}
		}
		if firstReadyClient == nil {
			return nil
		} else {
			pool.rrLastIndexes[clientType] = firstReadyClient.clientIdx
			return firstReadyClient
		}
	}

	return readyClients[0]
}
