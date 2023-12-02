package clients

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethpandaops/minccino/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/minccino/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type ClientPool struct {
	consensusPool *consensus.Pool
	executionPool *execution.Pool
	clients       []*PoolClient
}

type PoolClient struct {
	Config          *ClientConfig
	ConsensusClient *consensus.Client
	ExecutionClient *execution.Client
}

type ClientConfig struct {
	Name             string            `yaml:"name"`
	ConsensusUrl     string            `yaml:"consensusUrl"`
	ConsensusHeaders map[string]string `yaml:"consensusHeaders"`
	ExecutionUrl     string            `yaml:"executionUrl"`
	ExecutionHeaders map[string]string `yaml:"executionHeaders"`
}

func NewClientPool() (*ClientPool, error) {
	consensusPool, err := consensus.NewPool(&consensus.PoolConfig{})
	if err != nil {
		return nil, fmt.Errorf("could not init consensus pool: %w", err)
	}
	executionPool, err := execution.NewPool(&execution.PoolConfig{})
	if err != nil {
		return nil, fmt.Errorf("could not init execution pool: %w", err)
	}
	return &ClientPool{
		consensusPool: consensusPool,
		executionPool: executionPool,
		clients:       make([]*PoolClient, 0),
	}, nil
}

func (pool *ClientPool) AddClient(config *ClientConfig) error {
	consensusClient, err := pool.consensusPool.AddEndpoint(&consensus.ClientConfig{
		Name:    config.Name,
		Url:     config.ConsensusUrl,
		Headers: config.ConsensusHeaders,
	})
	if err != nil {
		return fmt.Errorf("could not init consensus client: %w", err)
	}

	executionClient, err := pool.executionPool.AddEndpoint(&execution.ClientConfig{
		Name:    config.Name,
		Url:     config.ExecutionUrl,
		Headers: config.ExecutionHeaders,
	})
	if err != nil {
		return fmt.Errorf("could not init consensus client: %w", err)
	}

	poolClient := &PoolClient{
		Config:          config,
		ConsensusClient: consensusClient,
		ExecutionClient: executionClient,
	}

	consensusClient.SubscribeBlockEvent(&consensus.Subscription[*consensus.Block]{
		Handler: func(block *consensus.Block) error {
			go pool.processConsensusBlockNotification(block, poolClient)
			return nil
		},
	})
	pool.clients = append(pool.clients, poolClient)
	return nil
}

func (pool *ClientPool) processConsensusBlockNotification(block *consensus.Block, poolClient *PoolClient) {
	versionedBlock := block.AwaitBlock(2 * time.Second)
	if versionedBlock == nil {
		logrus.Warnf("cl/el block notification failed: AwaitBlock timeout (client: %v, slot: %v, root: 0x%x)", poolClient.Config.Name, block.Slot, block.Root)
		return
	}
	hash, err := versionedBlock.ExecutionBlockHash()
	if err != nil {
		logrus.Warnf("cl/el block notification failed: %s (client: %v, slot: %v, root: 0x%x)", err, poolClient.Config.Name, block.Slot, block.Root)
		return
	}
	number, err := versionedBlock.ExecutionBlockNumber()
	if err != nil {
		logrus.Warnf("cl/el block notification failed: %s (client: %v, slot: %v, root: 0x%x)", err, poolClient.Config.Name, block.Slot, block.Root)
		return
	}
	poolClient.ExecutionClient.NotifyNewBlock(common.Hash(hash), number)
}
