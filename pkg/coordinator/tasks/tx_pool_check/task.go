package txpoolcheck

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
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
	latestBlock, err := executionClients[0].GetRPCClient().GetLatestBlock(ctx)
	if err != nil {
		t.logger.Errorf("Failed to fetch latest block: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	nonce, err := executionClients[0].GetRPCClient().GetEthClient().NonceAt(ctx, crypto.PubkeyToAddress(privKey.PublicKey), latestBlock.Number())
	if err != nil {
		t.logger.Errorf("Failed to fetch nonce: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	nonce++

	if t.config.Nonce != nil {
		t.logger.Infof("Using custom nonce: %d", *t.config.Nonce)
		nonce = *t.config.Nonce
	}

	t.logger.Infof("Starting nonce: %d", nonce)
	clientIndex := rand.Intn(len(executionClients))
	client := executionClients[clientIndex]

	t.logger.Infof("Using client: %s", client.GetName())

	var totalLatency time.Duration

	for i := 0; i < t.config.TxCount; i++ {
		tx, err := createDummyTransaction(nonce, chainID, privKey)
		if err != nil {
			t.logger.Errorf("Failed to create transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		startTx := time.Now()

		err = client.GetRPCClient().SendTransaction(ctx, tx)
		if err != nil {
			t.logger.Errorf("Failed to send transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		// wait for tx to be confirmed
		confirmed := false
		timeout := time.After(10 * time.Second)
		for !confirmed {
			select {
			case <-timeout:
				t.logger.Errorf("Timeout waiting for tx confirmation for tx: %s", tx.Hash().Hex())
				t.ctx.SetResult(types.TaskResultFailure)
				return fmt.Errorf("timeout waiting for tx confirmation")
			default:
				time.Sleep(50 * time.Millisecond)
				fetchedTx, _, err := client.GetRPCClient().GetEthClient().TransactionByHash(ctx, tx.Hash())
				if err != nil {
					// retry on error
					continue
				}
				if fetchedTx != nil {
					confirmed = true
				}
			}
		}
		
		nonce++

		latency := time.Since(startTx)
		totalLatency += latency

		if (i+1)%t.config.MeasureInterval == 0 {
			avgSoFar := totalLatency.Milliseconds() / int64(i+1)
			t.logger.Infof("Processed %d transactions, current avg latency: %dms", i+1, avgSoFar)
		}
	}

	// todo: change, cause the average latency isn't measured that way. It's a test only for the future percentiles measurement
	avgLatency := totalLatency / time.Duration(t.config.TxCount)
	t.logger.Infof("Average transaction latency: %dms", avgLatency.Milliseconds())

	if t.config.FailOnHighLatency && avgLatency.Milliseconds() > t.config.ExpectedLatency {
		t.logger.Errorf("Transaction latency too high: %dms (expected <= %dms)", avgLatency.Milliseconds(), t.config.ExpectedLatency)
		t.ctx.SetResult(types.TaskResultFailure)
	} else {
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.Outputs.SetVar("tx_count", t.config.TxCount)
		t.ctx.Outputs.SetVar("avg_latency_ms", avgLatency.Milliseconds())
	}

	// select random client, not the first
	client = executionClients[(clientIndex+1)%len(executionClients)]
	t.logger.Infof("Using second random client: %s", client.GetName())

	startTime := time.Now()
	sentTxCount := 0
	tx := ethtypes.Transaction{}

	for i := 0; i < t.config.TxCount; i++ {
		// generate and sign tx
		tx, err := createDummyTransaction(nonce, chainID, privKey)
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
		nonce++

		if sentTxCount%t.config.MeasureInterval == 0 {
			elapsed := time.Since(startTime)
			t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
		}
	}

	confirmed := false
	timeout := time.After(10 * time.Second)

	for !confirmed {
		select {
		case <-timeout:
			t.logger.Errorf("Timeout waiting for tx confirmation for tx: %s", tx.Hash().Hex())
			t.ctx.SetResult(types.TaskResultFailure)
			return fmt.Errorf("timeout waiting for tx confirmation")
		// only the last transaction is checked, when the loop is done
		default:
			time.Sleep(50 * time.Millisecond)
			fetchedTx, _, err := client.GetRPCClient().GetEthClient().TransactionByHash(ctx, tx.Hash())
			if err != nil {
				// retry on error
				continue
			}
			if fetchedTx != nil {
				confirmed = true
			}
		}
	}

	totalTime := time.Since(startTime)
	t.logger.Infof("Total time for %d transactions: %.2fs", sentTxCount, totalTime.Seconds())
	t.ctx.Outputs.SetVar("total_time_ms", totalTime.Milliseconds())

	return nil
}

func createDummyTransaction(nonce uint64, chainID *big.Int, privateKey *ecdsa.PrivateKey) (*ethtypes.Transaction, error) {
	// create a dummy transaction, we don't care about the actual data
	toAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	tx := ethtypes.NewTx(&ethtypes.LegacyTx{
		// Nonce:    nonce,
		To:       &toAddress,
		Value:    big.NewInt(100),
		Gas:      21000,
		GasPrice: big.NewInt(1),
		// random data + nonce to hex
		Data: 		[]byte(fmt.Sprintf("0xdeadbeef%v", nonce)),
	})

	signer := ethtypes.LatestSignerForChainID(chainID)
	signedTx, err := ethtypes.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}
