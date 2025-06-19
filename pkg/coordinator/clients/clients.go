package clients

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/consensus"
	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

type ClientPool struct {
	logger        logrus.FieldLogger
	ctx           context.Context
	ctxCancel     context.CancelFunc
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
	ConsensusURL     string            `yaml:"consensusUrl"`
	ConsensusHeaders map[string]string `yaml:"consensusHeaders"`
	ExecutionURL     string            `yaml:"executionUrl"`
	ExecutionHeaders map[string]string `yaml:"executionHeaders"`
}

func NewClientPool(logger logrus.FieldLogger) (*ClientPool, error) {
	return NewClientPoolWithContext(context.Background(), logger)
}

func NewClientPoolWithContext(ctx context.Context, logger logrus.FieldLogger) (*ClientPool, error) {
	poolCtx, ctxCancel := context.WithCancel(ctx)

	consensusPool, err := consensus.NewPool(poolCtx, &consensus.PoolConfig{
		FollowDistance: 10,
		ForkDistance:   1,
	}, logger.WithField("module", "consensus"))
	if err != nil {
		ctxCancel()
		return nil, fmt.Errorf("could not init consensus pool: %w", err)
	}

	executionPool, err := execution.NewPool(poolCtx, &execution.PoolConfig{
		FollowDistance: 10,
		ForkDistance:   1,
	}, logger.WithField("module", "execution"))
	if err != nil {
		ctxCancel()
		return nil, fmt.Errorf("could not init execution pool: %w", err)
	}

	return &ClientPool{
		logger:        logger.WithField("module", "clients"),
		ctx:           poolCtx,
		ctxCancel:     ctxCancel,
		consensusPool: consensusPool,
		executionPool: executionPool,
		clients:       make([]*PoolClient, 0),
	}, nil
}

func (pool *ClientPool) Close() {
	pool.ctxCancel()
}

func (pool *ClientPool) AddClient(config *ClientConfig) error {
	consensusClient, err := pool.consensusPool.AddEndpoint(&consensus.ClientConfig{
		Name:    config.Name,
		URL:     config.ConsensusURL,
		Headers: config.ConsensusHeaders,
	})
	if err != nil {
		return fmt.Errorf("could not init consensus client: %w", err)
	}

	executionClient, err := pool.executionPool.AddEndpoint(&execution.ClientConfig{
		Name:    config.Name,
		URL:     config.ExecutionURL,
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

	go pool.processConsensusBlockNotification(poolClient)

	pool.clients = append(pool.clients, poolClient)

	return nil
}

func (pool *ClientPool) processConsensusBlockNotification(poolClient *PoolClient) {
	defer func() {
		if err := recover(); err != nil {
			var err2 error
			if errval, errok := err.(error); errok {
				err2 = errval
			}

			pool.logger.WithError(err2).Errorf("uncaught panic in processConsensusBlockNotification subroutine: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	subscription := poolClient.ConsensusClient.SubscribeBlockEvent(100)
	defer subscription.Unsubscribe()

	for {
		select {
		case <-pool.ctx.Done():
			return
		case block := <-subscription.Channel():
			versionedBlock := block.AwaitBlock(context.Background(), 2*time.Second)
			if versionedBlock == nil {
				pool.logger.Warnf("cl/el block notification failed: AwaitBlock timeout (client: %v, slot: %v, root: 0x%x)", poolClient.Config.Name, block.Slot, block.Root)
				break
			}

			hash, err := versionedBlock.ExecutionBlockHash()
			if err != nil {
				pool.logger.Warnf("cl/el block notification failed: %s (client: %v, slot: %v, root: 0x%x)", err, poolClient.Config.Name, block.Slot, block.Root)
				break
			}

			number, err := versionedBlock.ExecutionBlockNumber()
			if err != nil {
				pool.logger.Warnf("cl/el block notification failed: %s (client: %v, slot: %v, root: 0x%x)", err, poolClient.Config.Name, block.Slot, block.Root)
				break
			}

			poolClient.ExecutionClient.NotifyNewBlock(common.Hash(hash), number)
		}
	}
}

func (pool *ClientPool) GetConsensusPool() *consensus.Pool {
	return pool.consensusPool
}

func (pool *ClientPool) GetExecutionPool() *execution.Pool {
	return pool.executionPool
}

func (pool *ClientPool) GetAllClients() []*PoolClient {
	clients := make([]*PoolClient, len(pool.clients))
	copy(clients, pool.clients)

	return clients
}

func (pool *ClientPool) GetClientsByNamePatterns(includePattern, excludePattern string) []*PoolClient {
	clients := []*PoolClient{}
	for _, client := range pool.clients {
		if includePattern != "" {
			matched, _ := regexp.MatchString(includePattern, client.Config.Name)
			if !matched {
				continue
			}
		}

		if excludePattern != "" {
			matched, _ := regexp.MatchString(excludePattern, client.Config.Name)
			if matched {
				continue
			}
		}

		clients = append(clients, client)
	}

	for i, v := range rand.Perm(len(clients)) {
		clients[v], clients[i] = clients[i], clients[v]
	}

	return clients
}
