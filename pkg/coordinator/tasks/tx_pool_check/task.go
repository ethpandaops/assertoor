package txpoolcheck

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "tx_pool_check"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the throughput and latency of transactions in the Ethereum TxPool",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx      *types.TaskContext
	options  *types.TaskOptions
	config   Config
	logger   logrus.FieldLogger
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

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	executionClients := clientPool.GetExecutionPool().GetReadyEndpoints(true)

	if len(executionClients) == 0 {
		t.logger.Error("No execution clients available")
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.logger.Infof("Testing TxPool with %d transactions", t.config.TxCount)

	startTime := time.Now()
	sentTxCount := 0

	for i := 0; i < t.config.TxCount; i++ {
		client := executionClients[i%len(executionClients)]

		tx := createDummyTransaction(uint64(i))
		err := client.GetRPCClient().SendTransaction(ctx, tx)

		if err != nil {
			t.logger.WithField("client", client.GetName()).Errorf("Failed to send transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		sentTxCount++

		if sentTxCount%t.config.MeasureInterval == 0 {
			elapsed := time.Since(startTime)
			t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
		}
	}

	totalTime := time.Since(startTime)
	t.logger.Infof("Total time for %d transactions: %.2fs", sentTxCount, totalTime.Seconds())

	avgLatency := totalTime.Milliseconds() / int64(t.config.TxCount)
	t.logger.Infof("Average transaction latency: %dms", avgLatency)

	if t.config.FailOnHighLatency && avgLatency > t.config.ExpectedLatency {
		t.logger.Errorf("Transaction latency too high: %dms (expected <= %dms)", avgLatency, t.config.ExpectedLatency)
		t.ctx.SetResult(types.TaskResultFailure)
	} else {
		t.ctx.SetResult(types.TaskResultSuccess)
	}

	return nil
}

func createDummyTransaction(nonce uint64) *execution.Transaction {
	return &execution.Transaction{
		Nonce:    nonce,
		GasPrice: big.NewInt(1),
		GasLimit: 21000,
		To:       &common.Address{},
		Value:    big.NewInt(100),
	}
}
