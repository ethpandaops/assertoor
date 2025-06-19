package checkethcall

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
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
	ctx           *types.TaskContext
	options       *types.TaskOptions
	config        Config
	logger        logrus.FieldLogger
	ignoreResults [][]byte
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

	t.ignoreResults = make([][]byte, len(config.IgnoreResults))
	for i, ignoreResult := range config.IgnoreResults {
		t.ignoreResults[i] = common.FromHex(ignoreResult)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	executionPool := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool()

	// Subscribe to block events
	blockSubscription := executionPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	// Get the latest block
	var latestBlock *execution.Block

	for _, block := range executionPool.GetBlockCache().GetCachedBlocks() {
		if latestBlock == nil || block.Number > latestBlock.Number {
			latestBlock = block
		}
	}

	// Run the check with the latest block
	if latestBlock != nil {
		if t.config.BlockNumber == 0 {
			t.runCheck(ctx, latestBlock.Number, latestBlock)
		} else if latestBlock.Number >= t.config.BlockNumber {
			// if the block we're looking for already passed, run the check immediately and return
			t.runCheck(ctx, t.config.BlockNumber, nil)
			return nil
		}
	}

	// Listen for new blocks and run the check for all blocks (without a specific block number)
	// or once when the block we're looking for is reached
	for {
		select {
		case block := <-blockSubscription.Channel():
			if t.config.BlockNumber == 0 {
				// Run the check for all blocks
				t.runCheck(ctx, block.Number, block)
			} else if block.Number >= t.config.BlockNumber {
				// Run the check once for the block we're looking for
				if block.Number != t.config.BlockNumber {
					// already passed the block we're looking for, run the check without block reference
					block = nil
				}

				t.runCheck(ctx, t.config.BlockNumber, block)

				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) runCheck(ctx context.Context, blockNumber uint64, block *execution.Block) {
	// Set up the call message
	address := common.HexToAddress(t.config.CallAddress)
	callMsg := &ethereum.CallMsg{
		Data: common.FromHex(t.config.EthCallData),
		To:   &address,
	}

	// Get the client pool from the scheduler
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	// Get matching the clients from the pool
	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints(true)
		if len(clients) == 0 {
			t.logger.Error("check failed: no matching clients found")
			t.ctx.SetResult(types.TaskResultFailure)

			return
		}
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			t.logger.Error("check failed: no matching clients found with pattern %v", t.config.ClientPattern)
			t.ctx.SetResult(types.TaskResultFailure)

			return
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	var callResult []byte

	// Send the eth_call to all the clients
	awaitCtx, cancelAwait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelAwait()

	checkedClients := 0

	for i := 0; i < len(clients); i++ {
		client := clients[i]

		if block != nil && !block.AwaitSeenBy(awaitCtx, client) {
			t.logger.WithField("client", client.GetName()).Errorf("check failed: client did not see block #%v (%v)", block.Number, block.Hash.String())
			continue
		}

		// Log the client name
		t.logger.Infof("sending ethCall to client %v", client.GetName())

		// Send the eth_call
		blockBigNumber := big.NewInt(0).SetUint64(blockNumber)
		fetchedResult, err := client.GetRPCClient().GetEthCall(ctx, callMsg, blockBigNumber)

		// Check if the eth_call was successful
		if err != nil {
			t.logger.WithField("client", client.GetName()).Errorf("RPC error when sending eth_call %v: %v", callMsg, err)
			t.ctx.SetResult(types.TaskResultFailure)

			return
		}

		ignoreResult := t.ignoreResult(fetchedResult)

		// Check if the fetched result is the same as the result from previous client
		if callResult == nil {
			callResult = fetchedResult
			t.ctx.Outputs.SetVar("callResult", fmt.Sprintf("0x%x", fetchedResult))
		} else if !bytes.Equal(callResult, fetchedResult) {
			if ignoreResult {
				t.logger.WithField("client", client.GetName()).Infof("eth_call results mismatch against other client (got: 0x%x, expected: 0x%x, ignored value)", fetchedResult, callResult)
			} else {
				t.logger.WithField("client", client.GetName()).Errorf("eth_call results mismatch against other client (got: 0x%x, expected: 0x%x)", fetchedResult, callResult)
			}

			if t.config.FailOnMismatch && !ignoreResult {
				t.ctx.SetResult(types.TaskResultFailure)
			} else {
				t.ctx.SetResult(types.TaskResultNone)
			}

			return
		}

		// Check if the fetched result is the expected result
		if t.config.ExpectResult != "" {
			expectedBytes := common.FromHex(t.config.ExpectResult)
			if !bytes.Equal(fetchedResult, expectedBytes) {
				if ignoreResult {
					t.logger.WithField("client", client.GetName()).Infof("eth_call results mismatch against expected result (got 0x%x, expected: 0x%x, ignored value)", fetchedResult, expectedBytes)
				} else {
					t.logger.WithField("client", client.GetName()).Errorf("eth_call results mismatch against expected result (got 0x%x, expected: 0x%x)", fetchedResult, expectedBytes)
				}

				if t.config.FailOnMismatch && !ignoreResult {
					t.ctx.SetResult(types.TaskResultFailure)
				} else {
					t.ctx.SetResult(types.TaskResultNone)
				}

				return
			}
		}

		t.logger.Infof("eth_call to client %v was successful, ethCallResult: 0x%x", client.GetName(), fetchedResult)

		checkedClients++
	}

	if checkedClients > 0 {
		t.ctx.SetResult(types.TaskResultSuccess)
	}
}

func (t *Task) ignoreResult(result []byte) bool {
	for _, ignoreResult := range t.ignoreResults {
		if bytes.Equal(result, ignoreResult) {
			return true
		}
	}

	return false
}
