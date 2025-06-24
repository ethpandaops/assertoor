package txloadtool

import (
	"fmt"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// LoadResult represents the result of a load test
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
func NewLoadResult(totNumberOfTxs int) *LoadResult {
	return &LoadResult{
		TotalTxs:                      totNumberOfTxs,
		Txs:                           make([]*ethtypes.Transaction, totNumberOfTxs),
		TxStartTime:                   make([]time.Time, totNumberOfTxs),
		LatenciesMus:                  make([]int64, totNumberOfTxs),
		SentTxCount:                   0,
		DuplicatedP2PEventCount:       0,
		CoordinatedOmissionEventCount: 0,
	}
}

// Load represents a load test that generates and sends transactions at a specified TPS (transactions per second) level
type Load struct {
	target       *LoadTarget
	testDeadline time.Time
	TPS          int
	DurationS    int
	LogInterval  int
	Result       *LoadResult
}

// NewLoad creates a new Load instance
func NewLoad(target *LoadTarget, tps, durationS int, testDeadline time.Time, logInterval int) *Load {
	return &Load{
		target:       target,
		TPS:          tps,
		DurationS:    durationS,
		testDeadline: testDeadline,
		LogInterval:  logInterval,
		Result:       NewLoadResult(tps * durationS),
	}
}

// Execute the load test generating and sending transactions at the TPS and duration specified in the Load struct.
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
		endTime := l.Result.StartTime.Add(time.Second * time.Duration(l.DurationS))

		if l.testDeadline.Before(endTime) {
			l.testDeadline = endTime
		}

		l.target.logger.Infof("Starting transaction generation at %s", l.Result.StartTime)

		// Create a ticker to maintain consistent timing
		interval := time.Second / time.Duration(l.TPS)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Generate and send transactions
		for i := 0; i < l.Result.TotalTxs; i++ {
			// Wait for the next tick to maintain QPS
			<-ticker.C

			// Generate and send tx
			before := time.Now()

			go l.generateAndSendTx(i)

			after := time.Now()

			// Check coordinated omission
			duration := after.Sub(before)
			if duration > interval {
				l.Result.CoordinatedOmissionEventCount++
			}

			// log every l.LogInterval
			if (i+1)%l.LogInterval == 0 {
				l.target.logger.Infof("Generated %d transactions in %.2fs", i+1, time.Since(l.Result.StartTime).Seconds())
			}

			select {
			case <-l.target.ctx.Done():
				l.target.logger.Warnf("Task cancelled, stopping transaction generation.")
				return
			default:
				// check if the execution failed
				if l.Result.Failed {
					return
				}
				// if testDeadline reached, stop sending txs
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

func (l *Load) generateAndSendTx(i int) {
	tx, err := l.target.GenerateTransaction(i)
	if err != nil {
		l.target.logger.Errorf("Failed to create transaction: %v", err)
		l.target.taskCtx.SetResult(types.TaskResultFailure)
		l.Result.Failed = true

		return
	}

	l.Result.TxStartTime[i] = time.Now()
	err = l.target.SendTransaction(tx)

	if err != nil {
		if !l.Result.Failed {
			l.target.logger.WithField("node", l.target.node.GetName()).Errorf("Failed to send transaction: %v", err)
			l.target.taskCtx.SetResult(types.TaskResultFailure)
			l.Result.Failed = true
		}

		return
	}

	l.Result.Txs[i] = tx
	l.Result.SentTxCount++

	// log transaction sending
	if l.Result.SentTxCount%l.LogInterval == 0 {
		elapsed := time.Since(l.Result.StartTime)
		l.target.logger.Infof("Sent %d transactions in %.2fs", l.Result.SentTxCount, elapsed.Seconds())
	}
}

// MeasurePropagationLatencies reads P2P events and calculates propagation latencies for each transaction
func (l *Load) MeasurePropagationLatencies() (*LoadResult, error) {
	// Get a P2P connection to read events
	peer := NewPeer(l.target.ctx, l.target.taskCtx, l.target.logger, l.target.node)

	err := peer.Connect()
	if err != nil {
		l.target.logger.Errorf("Failed to get P2P connection: %v", err)
		l.target.taskCtx.SetResult(types.TaskResultFailure)

		return l.Result, fmt.Errorf("measurement stopped: failed to get P2P connection")
	}

	defer peer.Close()

	// Wait P2P event messages
	var receivedEvents = 0

	for {
		txs, err := peer.ReadTransactionMessages(time.Duration(60) * time.Second)
		if err != nil {
			if err.Error() == "timeoutExpired" {
				l.target.logger.Warnf("Timeout expired while reading p2p events")
				break
			}

			l.target.logger.Errorf("Failed reading p2p events: %v", err)
			l.target.taskCtx.SetResult(types.TaskResultFailure)
			l.Result.Failed = true

			return l.Result, fmt.Errorf("measurement stopped: failed reading p2p events")
		}

		if txs == nil || len(*txs) == 0 {
			l.target.logger.Warnf("No p2p events received")
			continue
		}

		for i, tx := range *txs {
			txData := tx.Data()
			// read txData that is in the format "txIndex:<index>"
			var txIndex int

			_, err := fmt.Sscanf(string(txData), "txIndex:%d", &txIndex)
			if err != nil {
				l.target.logger.Errorf("Failed to parse transaction data: %v", err)
				l.target.taskCtx.SetResult(types.TaskResultFailure)
				l.Result.Failed = true

				return l.Result, fmt.Errorf("measurement stopped: failed to parse transaction data at event %d", i)
			}

			if txIndex < 0 || txIndex >= l.Result.TotalTxs {
				l.target.logger.Errorf("Transaction index out of range: %d", txIndex)
				l.target.taskCtx.SetResult(types.TaskResultFailure)
				l.Result.Failed = true

				return l.Result, fmt.Errorf("measurement stopped: transaction index out of range at event %d", i)
			}

			// log the duplicated p2p events and count duplicated p2p events
			if l.Result.LatenciesMus[txIndex] != 0 {
				l.Result.DuplicatedP2PEventCount++
			} else {
				l.Result.LastMeasureDelay = time.Since(l.Result.StartTime)
				l.Result.LatenciesMus[txIndex] = time.Since(l.Result.TxStartTime[txIndex]).Microseconds()
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
			l.Result.LatenciesMus[i] = -1
		}
	}

	if l.Result.NotReceivedP2PEventCount > 0 {
		l.target.logger.Warnf("Missed p2p events: %d", l.Result.NotReceivedP2PEventCount)
	}

	return l.Result, nil
}
