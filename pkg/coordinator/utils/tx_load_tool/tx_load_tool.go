package tx_load_tool

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/params"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/execution"
	"github.com/noku-team/assertoor/pkg/coordinator/helper"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/sentry"
	"github.com/noku-team/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

type LoadTool struct {
	ctx      context.Context
	task_ctx *types.TaskContext
	wallet   *wallet.Wallet
	logger   logrus.FieldLogger
	client   *execution.Client
	failed   bool
}

// NewLoadTool creates a new LoadTool instance
func NewLoadTool(ctx context.Context, task_ctx *types.TaskContext, logger logrus.FieldLogger,
	wallet *wallet.Wallet, client *execution.Client) *LoadTool {
	return &LoadTool{
		ctx:      ctx,
		task_ctx: task_ctx,
		wallet:   wallet,
		logger:   logger,
		client:   client,
	}
}

// ExecuteTPSLevel generates and sends transactions at the specified TPS level for the specified duration
func (t *LoadTool) ExecuteTPSLevel(TPS int, duration_s int, testDeadline time.Time) ([]*ethtypes.Transaction, []int64, int, int, int, bool) {
	// Prepare to collect transaction latencies
	const logInterval = 100 // Log every 100 transactions
	var totNumberOfTxes int = TPS * duration_s
	var txs []*ethtypes.Transaction = make([]*ethtypes.Transaction, totNumberOfTxes)
	var txStartTime []time.Time = make([]time.Time, totNumberOfTxes)
	var latenciesMus = make([]int64, totNumberOfTxes)

	startTime := time.Now()
	isFailed := false
	sentTxCount := 0
	duplicatedP2PEventCount := 0
	coordinatedOmissionEventCount := 0

	conn, err := t.getTcpConn(t.ctx, t.client)
	if err != nil {
		t.logger.Errorf("Failed to get wire eth TCP connection: %v", err)
		t.task_ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	defer conn.Close()

	// Start generating and sending transactions
	go func() {
		startExecTime := time.Now()
		endTime := startExecTime.Add(time.Second * time.Duration(duration_s))

		// Generate and send transactions
		for i := 0; i < totNumberOfTxes; i++ {
			// Calculate how much time we have left
			remainingTime := time.Until(endTime)

			// Calculate sleep time to distribute remaining transactions evenly
			sleepTime := remainingTime / time.Duration(totNumberOfTxes-i)

			// generate and send tx
			go func(i int) {

				tx, err := t.generateTransaction(t.ctx, i)
				if err != nil {
					t.logger.Errorf("Failed to create transaction: %v", err)
					t.task_ctx.SetResult(types.TaskResultFailure)
					isFailed = true
					return
				}

				txStartTime[i] = time.Now()
				err = t.client.GetRPCClient().SendTransaction(t.ctx, tx)
				if err != nil {
					t.logger.WithField("client", t.client.GetName()).Errorf("Failed to send transaction: %v", err)
					t.task_ctx.SetResult(types.TaskResultFailure)
					isFailed = true
					return
				}

				txs[i] = tx
				sentTxCount++

				// log transaction sending
				if sentTxCount%logInterval == 0 {
					elapsed := time.Since(startTime)
					t.logger.Infof("Sent %d transactions in %.2fs", sentTxCount, elapsed.Seconds())
				}

			}(i)

			// Sleep to control the TPS
			if i < totNumberOfTxes-1 {
				if sleepTime > 0 {
					time.Sleep(sleepTime)
				} else {
					coordinatedOmissionEventCount++
				}
			}

			select {
			case <-t.ctx.Done():
				t.logger.Warnf("Task cancelled, stopping transaction generation.")
				return
			default:
				// if testDeadline reached, stop sending txes
				if isFailed {
					return
				}
				if time.Now().After(testDeadline) {
					t.logger.Infof("Reached duration limit, stopping transaction generation.")
					return
				}
			}
		}
	}()

	// Wait P2P event messages
	var receivedEvents int = 0
	for {
		txes, err := conn.ReadTransactionMessages()
		if err != nil {
			t.logger.Errorf("Failed reading p2p events: %v", err)
			t.task_ctx.SetResult(types.TaskResultFailure)
			isFailed = true
			return
		}

		for _, tx := range *txes {
			tx_data := tx.Data()
			// read tx_data that is in the format "tx_index:<index>"
			var tx_index int
			_, err := fmt.Sscanf(string(tx_data), "tx_index:%d", &tx_index)
			if err != nil {
				t.logger.Errorf("Failed to parse transaction data: %v", err)
				t.task_ctx.SetResult(types.TaskResultFailure)
				isFailed = true
				return
			}
			if tx_index < 0 || tx_index >= totNumberOfTxes {
				t.logger.Errorf("Transaction index out of range: %d", tx_index)
				t.task_ctx.SetResult(types.TaskResultFailure)
				isFailed = true
				return
			}

			// log the duplicated p2p events, and count duplicated p2p events
			// todo: add a timeout of N seconds that activates if duplicatedP2PEventCount + receivedEvents >= totNumberOfTxes, if exceeded, exit the function
			if latenciesMus[tx_index] != 0 {
				duplicatedP2PEventCount++
			}

			latenciesMus[tx_index] = time.Since(txStartTime[tx_index]).Microseconds()
			receivedEvents++

			if receivedEvents%logInterval == 0 {
				t.logger.Infof("Received %d p2p events", receivedEvents)
			}
		}

		if receivedEvents >= totNumberOfTxes {
			t.logger.Infof("Reading of p2p events finished")
			break
		}

		select {
		case <-t.ctx.Done():
			t.logger.Warnf("Task cancelled, stopping reading p2p events.")
			return
		default:
			// check test deadline
			if time.Now().After(testDeadline) {
				t.logger.Warnf("Reached duration limit, stopping reading p2p events.")
				return
			}
		}
	}

	lastMeasureDelay := time.Since(startTime)
	t.logger.Infof("Last measure delay since start time: %s", lastMeasureDelay)

	if coordinatedOmissionEventCount > 0 {
		t.logger.Warnf("Coordinated omission events: %d", coordinatedOmissionEventCount)
	}

	if duplicatedP2PEventCount > 0 {
		t.logger.Warnf("Duplicated p2p events: %d", duplicatedP2PEventCount)
	}

	// Check if we received all transactions p2p events
	notReceivedP2PEventCount := 0
	for i := 0; i < totNumberOfTxes; i++ {
		if latenciesMus[i] == 0 {
			notReceivedP2PEventCount++
			// Assign a default value for missing P2P events
			latenciesMus[i] = (time.Duration(duration_s) * time.Second).Microseconds()
		}
	}
	if notReceivedP2PEventCount > 0 {
		t.logger.Warnf("Missed p2p events: %d (assigned latency=duration)", notReceivedP2PEventCount)
	}

	return txs, latenciesMus, duplicatedP2PEventCount, coordinatedOmissionEventCount, notReceivedP2PEventCount, isFailed
}

func (t *LoadTool) getTcpConn(ctx context.Context, client *execution.Client) (*sentry.Conn, error) {
	chainConfig := params.AllDevChainProtocolChanges

	head, err := client.GetRPCClient().GetLatestBlock(ctx)
	if err != nil {
		t.task_ctx.SetResult(types.TaskResultFailure)
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
		t.task_ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	conn, err := sentry.GetTcpConn(client)
	if err != nil {
		t.logger.Errorf("Failed to get TCP connection: %v", err)
		t.task_ctx.SetResult(types.TaskResultFailure)
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

func (t *LoadTool) generateTransaction(ctx context.Context, i int) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		addr := t.wallet.GetAddress()
		toAddr := &addr

		txAmount, _ := crand.Int(crand.Reader, big.NewInt(0).SetUint64(10*1e18))

		feeCap := &helper.BigInt{Value: *big.NewInt(100000000000)} // 100 Gwei
		tipCap := &helper.BigInt{Value: *big.NewInt(1000000000)}   // 1 Gwei

		txObj := &ethtypes.DynamicFeeTx{
			ChainID:   t.task_ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: &tipCap.Value,
			GasFeeCap: &feeCap.Value,
			Gas:       50000,
			To:        toAddr,
			Value:     txAmount,
			Data:      []byte(fmt.Sprintf("tx_index:%d", i)),
		}

		return ethtypes.NewTx(txObj), nil
	})

	if err != nil {
		return nil, err
	}

	return tx, nil
}
