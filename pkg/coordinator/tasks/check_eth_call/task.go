package checkethcall

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"math/big"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_eth_call"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the response of an eth_call transaction",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	t.logger.Info("Checking eth_call...")
	// Log all the parameters sent to it
	t.logger.Infof("CallAddress: %v", t.config.CallAddress)
	t.logger.Infof("EthCallData: %v", t.config.EthCallData)
	t.logger.Infof("ExpectResult: %v", t.config.ExpectResult)

	// Sleep so that we move ahead one slot
	t.logger.Info("Sleeping for 20 seconds to move ahead atleast one slot")
	time.Sleep(20 * time.Second)
	t.logger.Info("Woke up after 20 seconds")

	var clients []*execution.Client
	var callMsg ethereum.CallMsg

	// Set up the call message
	callMsg.Data = common.FromHex(t.config.EthCallData)
	address := common.HexToAddress(t.config.CallAddress)
	callMsg.To = &address

	// Get the latest block
	blockNumber, err := t.getLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("error getting latest block: %v", err)
	}

	// Get the client pool from the scheduler
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	// Get all the clients from the pool
	poolClients := clientPool.GetAllClients()
	if len(poolClients) == 0 {
		return fmt.Errorf("no client found in pool")
	}
	// Get the execution clients from the pool clients
	clients = make([]*execution.Client, len(poolClients))
	for i, c := range poolClients {
		clients[i] = c.ExecutionClient
	}

	if len(clients) == 0 {
		return fmt.Errorf("no healthy clients found")
	} else {
		// Send the eth_call to all the clients
		for i := 0; i < len(clients); i++ {
			client := clients[i]

			// Log the client name
			t.logger.Infof("sending ethCall to client %v", client.GetName())

			// Initialize fetchedResult to zero
			var fetchedResult []byte

			// Send the eth_call
			fetchedResult, err = client.GetRPCClient().GetEthCall(ctx, callMsg, blockNumber)
			// Check if the eth_call was successful
			if err != nil {
				return fmt.Errorf("RPC error when sending ethCall %v: %v to client %v", callMsg, err, client.GetName())
			} else if len(fetchedResult) == 0 {
				return fmt.Errorf("RPC error when sending ethCall %v: %v to client %v", callMsg, err, client.GetName())
			}
			// Check if the fetched result is the expected result
			if common.Hash(fetchedResult).Hex() != t.config.ExpectResult {
				return fmt.Errorf("expected result not found, expected: %v, got: %v", t.config.ExpectResult, common.Hash(fetchedResult).Hex())
			}

			t.logger.Infof("ethCall to client %v was successful, ethCallResult: %v and expectedResult: %v", client.GetName(), common.Hash(fetchedResult), t.config.ExpectResult)
		}
	}

	return nil
}

func (t *Task) getLatestBlockNumber(ctx context.Context) (*big.Int, error) {

	// Get the client pool from the scheduler
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	// Get the latest block from the execution pool
	blocks := clientPool.GetExecutionPool().GetBlockCache().GetCachedBlocks()

	if len(blocks) == 0 || blocks[0] == nil || blocks[0].GetBlock() == nil {
		t.logger.Error("No blocks found or the first block is nil")
		return nil, fmt.Errorf("no blocks found or the first block is nil")
	}

	// Get the head block
	block := blocks[0].GetBlock()

	t.logger.Infof("Fetched head block number %v for the ethCall parameter", block.Number())

	return block.Number(), nil
}
