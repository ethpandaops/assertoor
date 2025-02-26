package txpoolcheck

import (
	"context"
	"fmt"
	"math/big"
	"time"
	"crypto/ecdsa"
	"math/rand"

	"github.com/noku-team/assertoor/pkg/coordinator/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
	// get chain ID from execution client
	chainID, err := executionClients[0].GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		t.logger.Errorf("Failed to fetch chain ID: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.logger.Infof("Chain ID: %d", chainID)

	privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
	if err != nil {
		t.logger.Errorf("Failed to generate private key: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	// get last nonce
	nonce, err := executionClients[0].GetRPCClient().GetEthClient().PendingNonceAt(ctx, crypto.PubkeyToAddress(privKey.PublicKey))
	if err != nil {
		t.logger.Errorf("Failed to fetch nonce: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	t.logger.Infof("Starting nonce: %d", nonce)
	
	// select random client
	client := executionClients[rand.Intn(len(executionClients))]
	t.logger.Infof("Using client: %s", client.GetName())

	for i := 0; i < t.config.TxCount; i++ {
		// generate and sign tx
		tx, err := createDummyTransaction(uint64(i) + nonce, chainID, privKey)
		if err != nil {
			t.logger.Errorf("Failed to create transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		err = client.GetRPCClient().SendTransaction(ctx, tx)

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

func createDummyTransaction(nonce uint64, chainID *big.Int, privateKey *ecdsa.PrivateKey) (*ethtypes.Transaction, error) {
	// create a dummy transaction, we don't care about the actual data
	toAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	tx := ethtypes.NewTransaction(
		nonce,
		toAddress,
		big.NewInt(100),
		21000,
		big.NewInt(1),
		nil,
	)

	signer := ethtypes.LatestSignerForChainID(chainID)
	signedTx, err := ethtypes.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}