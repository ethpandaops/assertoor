package txpoollatencyanalysis

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/execution"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/sentry"
	txpool "github.com/noku-team/assertoor/pkg/coordinator/utils/tx_pool"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "tx_pool_latency_analysis"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the latency of transactions in the Ethereum TxPool",
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
	executionClients := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetReadyEndpoints(true)
	client := executionClients[rand.Intn(len(executionClients))]

	chainID, err := client.GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		t.logger.Errorf("Failed to fetch chain ID: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	conn, err := t.getTcpConn(ctx, client)
	if err != nil {
		t.logger.Errorf("Failed to get wire eth TCP connection: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	defer conn.Close()

	privKey, _ := crypto.HexToECDSA(t.config.PrivateKey)
	nonce, err := t.getNonce(ctx, privKey)
	if err != nil {
		t.logger.Errorf("Failed to fetch nonce: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	var totalLatency time.Duration
	var latencies []time.Duration
	retryCount := 0

	for i := 0; i < t.config.TxCount; i++ {
		tx, err := txpool.CreateDummyTransaction(nonce, chainID, privKey)
		if err != nil {
			t.logger.Errorf("Failed to create transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		startTx := time.Now()

		err = client.GetRPCClient().SendTransaction(ctx, tx)

		if err != nil {
			t.logger.Errorf("Failed to send transaction: %v. Nonce: %d. ", err, nonce)

			// retry increasing the nonce
			nonce++
			i--
			retryCount++

			if retryCount > 1000 {
				t.logger.Errorf("Too many retries")
				t.ctx.SetResult(types.TaskResultFailure)
				return nil
			}

			continue
		}

		retryCount = 0

		_, err = conn.ReadTransactionMessages()
		if err != nil {
			t.logger.Errorf("Failed to read transaction messages: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		nonce++

		latency := time.Since(startTx)
		latencies = append(latencies, latency)
		totalLatency += latency

		if (i+1)%t.config.MeasureInterval == 0 {
			avgSoFar := totalLatency.Milliseconds() / int64(i+1)
			t.logger.Infof("Processed %d transactions, current avg latency: %dms.", i+1, avgSoFar)
		}
	}

	avgLatency := totalLatency / time.Duration(t.config.TxCount)
	t.logger.Infof("Average transaction latency: %dms", avgLatency.Milliseconds())

	if t.config.FailOnHighLatency && avgLatency.Milliseconds() > t.config.ExpectedLatency {
		t.logger.Errorf("Transaction latency too high: %dms (expected <= %dms)", avgLatency.Milliseconds(), t.config.ExpectedLatency)
		t.ctx.SetResult(types.TaskResultFailure)
	} else {
		t.ctx.Outputs.SetVar("tx_count", t.config.TxCount)
		t.ctx.Outputs.SetVar("avg_latency_ms", avgLatency.Milliseconds())
		t.ctx.Outputs.SetVar("detailed_latencies", latencies)
		t.ctx.SetResult(types.TaskResultSuccess)
	}

	return nil
}

func (t *Task) getNonce(ctx context.Context, privKey *ecdsa.PrivateKey) (uint64, error) {
	executionClients := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetReadyEndpoints(true)

	head, err := executionClients[0].GetRPCClient().GetLatestBlock(ctx);
	if err != nil {
		return 0, err
	}

	nonce, err := executionClients[0].GetRPCClient().GetEthClient().NonceAt(ctx, crypto.PubkeyToAddress(privKey.PublicKey), head.Number())

	if t.config.Nonce != nil {
		t.logger.Infof("Using custom nonce: %d", *t.config.Nonce)
		nonce = *t.config.Nonce
	}

	return nonce, nil
}

func (t *Task) getTcpConn(ctx context.Context, client *execution.Client) (*sentry.Conn, error) {
	chainConfig := params.AllDevChainProtocolChanges;

	head, err := client.GetRPCClient().GetLatestBlock(ctx);
	if err != nil {
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	chainID, err := client.GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		return nil, err
	}

	chainConfig.ChainID = chainID

	genesis, err := client.GetRPCClient().GetEthClient().BlockByNumber(ctx, new(big.Int).SetUint64(0))
	if err != nil {
		t.logger.Errorf("Failed to fetch genesis block: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	conn, err := sentry.GetTcpConn(client)
	if err != nil {
		t.logger.Errorf("Failed to get TCP connection: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	forkId := forkid.NewID(chainConfig, genesis, head.NumberU64(), head.Time())

	// handshake
	err = conn.Peer(chainConfig.ChainID, genesis.Hash(), head.Hash(), forkId, nil)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
