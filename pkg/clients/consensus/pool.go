package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/gloas"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

type SchedulerMode uint8

var (
	RoundRobinScheduler SchedulerMode = 1
)

type PoolConfig struct {
	FollowDistance uint32 `yaml:"followDistance" envconfig:"CONSENSUS_POOL_FOLLOW_DISTANCE"`
	ForkDistance   uint32 `yaml:"forkDistance" envconfig:"CONSENSUS_POOL_FORK_DISTANCE"`
	SchedulerMode  string `yaml:"schedulerMode" envconfig:"CONSENSUS_POOL_SCHEDULER_MODE"`
}

type Pool struct {
	config         *PoolConfig
	ctx            context.Context
	logger         logrus.FieldLogger
	clientCounter  uint16
	clients        []*Client
	blockCache     *BlockCache
	forkCacheMutex sync.Mutex
	forkCache      map[int64][]*HeadFork

	schedulerMode  SchedulerMode
	schedulerMutex sync.Mutex
	rrLastIndexes  map[ClientType]uint16
}

func NewPool(ctx context.Context, config *PoolConfig, logger logrus.FieldLogger) (*Pool, error) {
	var err error

	pool := Pool{
		config:        config,
		ctx:           ctx,
		logger:        logger,
		clients:       make([]*Client, 0),
		forkCache:     map[int64][]*HeadFork{},
		rrLastIndexes: map[ClientType]uint16{},
	}

	switch config.SchedulerMode {
	case "", "rr", "roundrobin":
		pool.schedulerMode = RoundRobinScheduler
	default:
		return nil, fmt.Errorf("unknown pool schedulerMode: %v", config.SchedulerMode)
	}

	pool.blockCache, err = NewBlockCache(ctx, logger, config.FollowDistance)
	if err != nil {
		return nil, err
	}

	return &pool, nil
}

func (pool *Pool) GetBlockCache() *BlockCache {
	return pool.blockCache
}

func (pool *Pool) GetValidatorSet() map[phase0.ValidatorIndex]*v1.Validator {
	return pool.blockCache.getCachedValidatorSet(func() map[phase0.ValidatorIndex]*v1.Validator {
		client := pool.GetReadyEndpoint(AnyClient)
		if client == nil {
			pool.logger.Errorf("could not load validator set: no ready client")
			return nil
		}

		valset, err := client.GetRPCClient().GetStateValidators(client.clientCtx, "head")
		if err != nil {
			pool.logger.Errorf("could not load validator set: %v", err)
			return nil
		}

		return valset
	})
}

func (pool *Pool) GetBuilderSet() []*BuilderInfo {
	return pool.blockCache.getCachedBuilderSet(func() []*BuilderInfo {
		client := pool.GetReadyEndpoint(AnyClient)
		if client == nil {
			pool.logger.Errorf("could not load builder set: no ready client")
			return nil
		}

		state, err := client.GetRPCClient().GetState(client.clientCtx, "head")
		if err != nil {
			pool.logger.Errorf("could not load beacon state for builder set: %v", err)
			return nil
		}

		if state.Version < spec.DataVersionGloas || state.Gloas == nil {
			return nil
		}

		// Update validator set cache from the same state
		pool.updateValidatorSetFromState(state)

		builders := make([]*BuilderInfo, len(state.Gloas.Builders))
		for i, b := range state.Gloas.Builders {
			builders[i] = &BuilderInfo{
				Index:   gloas.BuilderIndex(i),
				Builder: b,
			}
		}

		return builders
	})
}

func (pool *Pool) updateValidatorSetFromState(state *spec.VersionedBeaconState) {
	validators, err := state.Validators()
	if err != nil || len(validators) == 0 {
		return
	}

	balances, err := state.ValidatorBalances()
	if err != nil {
		return
	}

	currentSlot, err := state.Slot()
	if err != nil {
		return
	}

	specs := pool.blockCache.GetSpecs()
	if specs == nil {
		return
	}

	currentEpoch := phase0.Epoch(uint64(currentSlot) / specs.SlotsPerEpoch)

	valset := make(map[phase0.ValidatorIndex]*v1.Validator, len(validators))
	for i, val := range validators {
		idx := phase0.ValidatorIndex(i)

		balance := phase0.Gwei(0)
		if i < len(balances) {
			balance = balances[i]
		}

		valset[idx] = &v1.Validator{
			Index:     idx,
			Balance:   balance,
			Status:    computeValidatorStatus(val, currentEpoch),
			Validator: val,
		}
	}

	pool.blockCache.SetValidatorSet(valset)
}

func computeValidatorStatus(val *phase0.Validator, epoch phase0.Epoch) v1.ValidatorState {
	farFuture := phase0.Epoch(0xFFFFFFFFFFFFFFFF)

	if val.ActivationEligibilityEpoch == farFuture {
		return v1.ValidatorStatePendingInitialized
	}

	if val.ActivationEpoch > epoch {
		return v1.ValidatorStatePendingQueued
	}

	if val.ExitEpoch > epoch {
		if val.Slashed {
			return v1.ValidatorStateActiveSlashed
		}

		if val.ExitEpoch == farFuture {
			return v1.ValidatorStateActiveOngoing
		}

		return v1.ValidatorStateActiveExiting
	}

	if val.WithdrawableEpoch > epoch {
		if val.Slashed {
			return v1.ValidatorStateExitedSlashed
		}

		return v1.ValidatorStateExitedUnslashed
	}

	if val.EffectiveBalance != 0 {
		return v1.ValidatorStateWithdrawalPossible
	}

	return v1.ValidatorStateWithdrawalDone
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

func (pool *Pool) AwaitReadyEndpoint(ctx context.Context, clientType ClientType) *Client {
	for {
		client := pool.GetReadyEndpoint(clientType)
		if client != nil {
			return client
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(1 * time.Second):
		}
	}
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

	if pool.schedulerMode == RoundRobinScheduler {
		var firstReadyClient *Client

		for _, client := range readyClients {
			if clientType != AnyClient && clientType != client.clientType {
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
		}

		pool.rrLastIndexes[clientType] = firstReadyClient.clientIdx

		return firstReadyClient
	}

	return readyClients[0]
}
