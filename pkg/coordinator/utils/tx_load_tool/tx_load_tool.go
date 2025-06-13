package tx_load_tool

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/execution"
	"github.com/noku-team/assertoor/pkg/coordinator/helper"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/sentry"
	"github.com/noku-team/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

type LoadTarget struct {
	ctx      context.Context
	task_ctx *types.TaskContext
	wallet   *wallet.Wallet
	logger   logrus.FieldLogger
	client   *execution.Client
}

// NewLoadTarget creates a new LoadTarget instance
func NewLoadTarget(ctx context.Context, task_ctx *types.TaskContext, logger logrus.FieldLogger,
	wallet *wallet.Wallet, client *execution.Client) *LoadTarget {
	return &LoadTarget{
		ctx:      ctx,
		task_ctx: task_ctx,
		wallet:   wallet,
		logger:   logger,
		client:   client,
	}
}

type LoadResult struct {
	Failed    bool
	StartTime time.Time
	EndTime   time.Time // sent time of the last transaction
	// Data collected during the load test
	TotalTxs         int
	Txs              []*ethtypes.Transaction
	TxStartTime      []time.Time
	LatenciesMus     []int64
	LastMeasureDelay time.Duration
	// Statistics
	SentTxCount                   int
	DuplicatedP2PEventCount       int
	CoordinatedOmissionEventCount int
	NotReceivedP2PEventCount      int
}

// NewLoadResult creates a new LoadResult instance
func NewLoadResult(totNumberOfTxes int) *LoadResult {
	return &LoadResult{
		TotalTxs:                      totNumberOfTxes,
		Txs:                           make([]*ethtypes.Transaction, totNumberOfTxes),
		TxStartTime:                   make([]time.Time, totNumberOfTxes),
		LatenciesMus:                  make([]int64, totNumberOfTxes),
		SentTxCount:                   0,
		DuplicatedP2PEventCount:       0,
		CoordinatedOmissionEventCount: 0,
	}
}

type Load struct {
	target          *LoadTarget
	testDeadline    time.Time
	TPS             int
	Duration_s      int
	LogInterval     int
	Result          *LoadResult
}

// NewLoad creates a new Load instance
func NewLoad(target *LoadTarget, TPS int, duration_s int, testDeadline time.Time, logInterval int) *Load {
	return &Load{
		target:          target,
		TPS:             TPS,
		Duration_s:      duration_s,
		testDeadline:    testDeadline,
		LogInterval:     logInterval,
		Result:          NewLoadResult(TPS * duration_s),
	}
}

// ExecuteTPSLevel generates and sends transactions at the specified TPS level for the specified duration
func (l *Load) Execute() error {
	// Prepare to collect transaction latencies
	l.Result.Failed = false
	l.Result.SentTxCount = 0
	l.Result.DuplicatedP2PEventCount = 0
	l.Result.CoordinatedOmissionEventCount = 0

	// Start generating and sending transactions
	go func() {
		// Sleep to ensure the start time is recorded correctly
		time.Sleep(100 * time.Millisecond)

		l.Result.StartTime = time.Now()
		endTime := l.Result.StartTime.Add(time.Second * time.Duration(l.Duration_s))
		l.target.logger.Infof("Starting transaction generation at %s", l.Result.StartTime)

		// Generate and send transactions
		for i := 0; i < l.Result.TotalTxs; i++ {
			// Calculate how much time we have left
			remainingTime := time.Until(endTime)

			// Calculate sleep time to distribute remaining transactions evenly
			sleepTime := remainingTime / time.Duration(l.Result.TotalTxs-i)

			// generate and send tx
			go func(i int) {

				tx, err := l.target.generateTransaction(i)
				if err != nil {
					l.target.logger.Errorf("Failed to create transaction: %v", err)
					l.target.task_ctx.SetResult(types.TaskResultFailure)
					l.Result.Failed = true
					return
				}

				l.Result.TxStartTime[i] = time.Now()
				err = l.target.sendTransaction(tx)
				if err != nil {
					l.target.logger.WithField("client", l.target.client.GetName()).Errorf("Failed to send transaction: %v", err)
					l.target.task_ctx.SetResult(types.TaskResultFailure)
					l.Result.Failed = true
					return
				}

				l.Result.Txs[i] = tx
				l.Result.SentTxCount++

				// log transaction sending
				if l.Result.SentTxCount%l.LogInterval == 0 {
					elapsed := time.Since(l.Result.StartTime)
					l.target.logger.Infof("Sent %d transactions in %.2fs", l.Result.SentTxCount, elapsed.Seconds())
				}

			}(i)

			// Sleep to control the TPS
			if i < l.Result.TotalTxs-1 {
				if sleepTime > 0 {
					time.Sleep(sleepTime)
				} else {
					l.Result.CoordinatedOmissionEventCount++
				}
			}

			select {
			case <-l.target.ctx.Done():
				l.target.logger.Warnf("Task cancelled, stopping transaction generation.")
				return
			default:
				// if testDeadline reached, stop sending txes
				if l.Result.Failed {
					return
				}
				if time.Now().After(l.testDeadline) {
					l.target.logger.Infof("Reached duration limit, stopping transaction generation.")
					return
				}
			}
		}

		l.Result.EndTime = time.Now()
		l.target.logger.Infof("Finished sending transactions at %s", l.Result.EndTime)
	}()

	return nil
}

// MeasurePropagationLatencies reads P2P events and calculates propagation latencies for each transaction
func (l *Load) MeasurePropagationLatencies() (*LoadResult, error) {

	// Get a P2P connection to read events
	conn, err := l.target.getTcpConn()
	if err != nil {
		l.target.logger.Errorf("Failed to get P2P connection: %v", err)
		l.target.task_ctx.SetResult(types.TaskResultFailure)
		return l.Result, fmt.Errorf("measurement stopped: failed to get P2P connection")
	}

	defer conn.Close()

	// Wait P2P event messages
	var receivedEvents int = 0
	for {
		txes, err := conn.ReadTransactionMessages(time.Duration(60) * time.Second)
		if err != nil {
			if err.Error() == "timeoutExpired" {
				l.target.logger.Warnf("Timeout expired while reading p2p events")
				break
			}

			l.target.logger.Errorf("Failed reading p2p events: %v", err)
			l.target.task_ctx.SetResult(types.TaskResultFailure)
			l.Result.Failed = true
			return l.Result, fmt.Errorf("measurement stopped: failed reading p2p events")
		}

		for i, tx := range *txes {
			tx_data := tx.Data()
			// read tx_data that is in the format "tx_index:<index>"
			var tx_index int
			_, err := fmt.Sscanf(string(tx_data), "tx_index:%d", &tx_index)
			if err != nil {
				l.target.logger.Errorf("Failed to parse transaction data: %v", err)
				l.target.task_ctx.SetResult(types.TaskResultFailure)
				l.Result.Failed = true
				return l.Result, fmt.Errorf("measurement stopped: failed to parse transaction data at event %d", i)
			}
			if tx_index < 0 || tx_index >= l.Result.TotalTxs {
				l.target.logger.Errorf("Transaction index out of range: %d", tx_index)
				l.target.task_ctx.SetResult(types.TaskResultFailure)
				l.Result.Failed = true
				return l.Result, fmt.Errorf("measurement stopped: transaction index out of range at event %d", i)
			}

			// log the duplicated p2p events, and count duplicated p2p events
			if l.Result.LatenciesMus[tx_index] != 0 {
				l.Result.DuplicatedP2PEventCount++
			} else {
				l.Result.LatenciesMus[tx_index] = time.Since(l.Result.TxStartTime[tx_index]).Microseconds()
				receivedEvents++
			}

			if receivedEvents%l.LogInterval == 0 {
				l.target.logger.Infof("Received %d p2p events", receivedEvents)
			}
		}

		if receivedEvents >= l.Result.TotalTxs {
			l.target.logger.Infof("Reading of p2p events finished")
			break
		}

		select {
		case <-l.target.ctx.Done():
			l.target.logger.Warnf("Task cancelled, stopping reading p2p events.")
			l.Result.Failed = true
			return l.Result, fmt.Errorf("measurement stopped: task cancelled")
		default:
			// check test deadline
			if time.Now().After(l.testDeadline) {
				l.target.logger.Warnf("Reached duration limit, stopping reading p2p events.")
				l.Result.Failed = true
				return l.Result, fmt.Errorf("measurement stopped: reached duration limit")
			}
			// check if the execution failed
			if l.Result.Failed {
				l.target.logger.Warnf("Execution failed, stopping reading p2p events.")
				return l.Result, fmt.Errorf("measurement stopped: execution failed")
			}
		}
	}

	// check if the execution failed
	if l.Result.Failed {
		l.target.logger.Warnf("Execution failed, stopping reading p2p events.")
		return l.Result, fmt.Errorf("measurement stopped: execution failed")
	}

	// Calculate the last measure delay
	l.Result.LastMeasureDelay = time.Since(l.Result.StartTime)
	l.target.logger.Infof("Last measure delay since start time: %s", l.Result.LastMeasureDelay)

	if l.Result.CoordinatedOmissionEventCount > 0 {
		l.target.logger.Warnf("Coordinated omission events: %d", l.Result.CoordinatedOmissionEventCount)
	}

	if l.Result.DuplicatedP2PEventCount > 0 {
		l.target.logger.Warnf("Duplicated p2p events: %d", l.Result.DuplicatedP2PEventCount)
	}

	// Check if we received all transactions p2p events
	l.Result.NotReceivedP2PEventCount = 0
	for i := 0; i < l.Result.TotalTxs; i++ {
		if l.Result.LatenciesMus[i] == 0 {
			l.Result.NotReceivedP2PEventCount++
			// Assign a default value for missing P2P events
			l.Result.LatenciesMus[i] = (time.Duration(l.Duration_s) * time.Second).Microseconds()
		}
	}
	if l.Result.NotReceivedP2PEventCount > 0 {
		l.target.logger.Warnf("Missed p2p events: %d (assigned latency=duration)", l.Result.NotReceivedP2PEventCount)
	}

	return l.Result, nil
}

func (t *LoadTarget) getTcpConn() (*sentry.Conn, error) {
	chainConfig := params.AllDevChainProtocolChanges

	head, err := t.client.GetRPCClient().GetLatestBlock(t.ctx)
	if err != nil {
		t.task_ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	chainID, err := t.client.GetRPCClient().GetEthClient().ChainID(t.ctx)
	if err != nil {
		return nil, err
	}

	chainConfig.ChainID = chainID

	genesis, err := t.client.GetRPCClient().GetEthClient().BlockByNumber(t.ctx, new(big.Int).SetUint64(0))
	if err != nil {
		t.logger.Errorf("Failed to fetch genesis block: %v", err)
		t.task_ctx.SetResult(types.TaskResultFailure)
		return nil, err
	}

	conn, err := sentry.GetTcpConn(t.client)
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

	t.logger.Infof("Connected to %s", t.client.GetName())

	return conn, nil
}

func (t *LoadTarget) generateTransaction(i int) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(t.ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
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

func (t *LoadTarget) sendTransaction(tx *ethtypes.Transaction) error {
	return t.client.GetRPCClient().SendTransaction(t.ctx, tx)
}
