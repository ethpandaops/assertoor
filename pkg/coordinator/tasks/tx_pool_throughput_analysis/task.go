package txpoolcheck

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
	TaskName       = "tx_pool_throughput_analysis"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the throughput of transactions in the Ethereum TxPool",
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
	chainID, err := executionClients[0].GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		t.logger.Errorf("Failed to fetch chain ID: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	client := executionClients[rand.Intn(len(executionClients))]

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

	startTime := time.Now()
	sentTxCount := 0

	go func() {
		for i := 0; i < t.config.TxCount; i++ {
			// generate and sign tx
			go func() {
					tx, err := txpool.CreateDummyTransaction(nonce, chainID, privKey)
					if err != nil {
						t.logger.Errorf("Failed to create transaction: %v", err)
						t.ctx.SetResult(types.TaskResultFailure)
						return
					}

					sentTxCount++
					nonce++

					err = client.GetRPCClient().SendTransaction(ctx, tx)

					if err != nil {
						t.logger.WithField("client", client.GetName()).Errorf("Failed to send transaction: %v", err)
						t.ctx.SetResult(types.TaskResultFailure)
						return
					}

					if sentTxCount%t.config.MeasureInterval == 0 {
						elapsed := time.Since(startTime)
						t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
					}
			}()

			// wait for 1/TxCount second: if 100 tx, than wait 10ms per cycle
			time.Sleep(time.Second / time.Duration(t.config.TxCount))
		}
	}()

	lastMeasureTime := time.Now()
	gotTx := 0

	for gotTx < t.config.TxCount {
			_, err := conn.ReadTransactionMessages()
			if err != nil {
				t.logger.Errorf("Failed to read transaction messages: %v", err)
				t.ctx.SetResult(types.TaskResultFailure)
				return nil
			}

			gotTx++

			if gotTx%t.config.MeasureInterval != 0 {
				continue
			}

			t.logger.Infof("Got %d transactions", gotTx)
			t.logger.Infof("Tx/s: (%d txs processed): %.2f / s \n", t.config.MeasureInterval, float64(t.config.MeasureInterval)*float64(time.Second)/float64(time.Since(lastMeasureTime)))

			lastMeasureTime = time.Now()
	}

	totalTime := time.Since(startTime)
	t.logger.Infof("Total time for %d transactions: %.2fs", sentTxCount, totalTime.Seconds())
	t.ctx.Outputs.SetVar("total_time_ms", totalTime.Milliseconds())
	t.ctx.SetResult(types.TaskResultSuccess)

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
	executionClients := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetReadyEndpoints(true)

	if len(executionClients) == 0 {
		return nil, fmt.Errorf("no execution clients available")
	}

	head, err := executionClients[0].GetRPCClient().GetLatestBlock(ctx);
	if err != nil {
		t.ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	chainID, err := executionClients[0].GetRPCClient().GetEthClient().ChainID(ctx)
	if err != nil {
		return nil, err
	}

	chainConfig.ChainID = chainID

	genesis, err := executionClients[0].GetRPCClient().GetEthClient().BlockByNumber(ctx, new(big.Int).SetUint64(0))
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

	t.logger.Infof("Connected to %s", client.GetName())

	return conn, nil
}
