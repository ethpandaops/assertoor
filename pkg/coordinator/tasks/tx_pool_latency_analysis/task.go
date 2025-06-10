package txpoollatencyanalysis

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/forkid"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/execution"
	"github.com/noku-team/assertoor/pkg/coordinator/helper"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/hdr"
	"github.com/noku-team/assertoor/pkg/coordinator/utils/sentry"
	"github.com/noku-team/assertoor/pkg/coordinator/wallet"
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
	wallet  *wallet.Wallet
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

	privKey, _ := crypto.HexToECDSA(config.PrivateKey)
	t.wallet, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	t.config = config
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	err := t.wallet.AwaitReady(ctx)
	if err != nil {
		return fmt.Errorf("cannot load wallet state: %w", err)
	}

	executionClients := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetReadyEndpoints(true)

	if len(executionClients) == 0 {
		t.logger.Errorf("No execution clients available")
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

	var totalLatency time.Duration
	var latencies []time.Duration

	var txs []*ethtypes.Transaction

	for i := 0; i < t.config.TxCount; i++ {
		tx, err := t.generateTransaction(ctx)
		if err != nil {
			t.logger.Errorf("Failed to create transaction: %v", err)
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		startTx := time.Now()

		err = client.GetRPCClient().SendTransaction(ctx, tx)
		if err != nil {
			t.logger.Errorf("Failed to send transaction: %v. Nonce: %d. ", err, tx.Nonce())
			t.ctx.SetResult(types.TaskResultFailure)
			return nil
		}

		txs = append(txs, tx)

		// Create a context with timeout for reading transaction messages
		readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			_, readErr := conn.ReadTransactionMessages()
			done <- readErr
		}()

		select {
		case err = <-done:
			if err != nil {
				t.logger.Errorf("Failed to read transaction messages: %v", err)
				t.ctx.SetResult(types.TaskResultFailure)
				return nil
			}
		case <-readCtx.Done():
			t.logger.Warnf("Timeout waiting for transaction message at index %d, retrying transaction", i)
			i-- // Retry this transaction
			continue
		}

		latency := time.Since(startTx)
		latencies = append(latencies, latency)
		totalLatency += latency

		if (i+1)%t.config.MeasureInterval == 0 {
			avgSoFar := totalLatency.Microseconds() / int64(i+1)
			t.logger.Infof("Processed %d transactions, current avg latency: %dmus.", i+1, avgSoFar)
		}
	}

	avgLatency := totalLatency / time.Duration(t.config.TxCount)
	t.logger.Infof("Average transaction latency: %dmus", avgLatency.Microseconds())

	// send to other clients, for speeding up tx mining
	for _, tx := range txs {
		for _, otherClient := range executionClients {
			if otherClient.GetName() == client.GetName() {
				continue
			}

			otherClient.GetRPCClient().SendTransaction(ctx, tx)
		}
	}

	// Convert latencies to microseconds for processing
	latenciesMus := make([]int64, len(latencies))
	for i, latency := range latencies {
		latenciesMus[i] = latency.Microseconds()
	}

	// Calculate statistics
	var totalLatencyMus int64
	var maxLatency int64 = 0
	var minLatency int64 = 0
	if len(latenciesMus) > 0 {
		minLatency = latenciesMus[0]
	}

	for _, lat := range latenciesMus {
		totalLatencyMus += lat
		if lat > maxLatency {
			maxLatency = lat
		}
		if lat < minLatency {
			minLatency = lat
		}
	}

	// Calculate mean
	var meanLatency float64 = 0
	if len(latenciesMus) > 0 {
		meanLatency = float64(totalLatencyMus) / float64(len(latenciesMus))
	}

	// Sort for percentiles
	sortedLatencies := make([]int64, len(latenciesMus))
	copy(sortedLatencies, latenciesMus)
	sort.Slice(sortedLatencies, func(i, j int) bool {
		return sortedLatencies[i] < sortedLatencies[j]
	})

	// Calculate percentiles
	percentile50th := float64(0)
	percentile90th := float64(0)
	percentile95th := float64(0)
	percentile99th := float64(0)

	if len(sortedLatencies) > 0 {
		getPercentile := func(pct float64) float64 {
			idx := int(float64(len(sortedLatencies)-1) * pct / 100)
			return float64(sortedLatencies[idx])
		}

		percentile50th = getPercentile(50)
		percentile90th = getPercentile(90)
		percentile95th = getPercentile(95)
		percentile99th = getPercentile(99)
	}

	// Create statistics map for output
	latenciesStats := map[string]float64{
		"total": float64(totalLatencyMus),
		"mean":  meanLatency,
		"50th":  percentile50th,
		"90th":  percentile90th,
		"95th":  percentile95th,
		"99th":  percentile99th,
		"max":   float64(maxLatency),
		"min":   float64(minLatency),
	}

	// Generate HDR plot
	plot, err := hdr.HdrPlot(latenciesMus)
	if err != nil {
		t.logger.Errorf("Failed to generate HDR plot: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)
		return nil
	}

	if t.config.FailOnHighLatency && avgLatency.Microseconds() > t.config.HighLatency {
		t.logger.Errorf("Transaction latency too high: %dmus (expected <= %dmus)", avgLatency.Microseconds(), t.config.HighLatency)
		t.ctx.SetResult(types.TaskResultFailure)
	} else {
		t.ctx.Outputs.SetVar("tx_count", t.config.TxCount)
		t.ctx.Outputs.SetVar("avg_latency_mus", avgLatency.Microseconds())
		t.ctx.Outputs.SetVar("latencies", latenciesStats)

		t.ctx.SetResult(types.TaskResultSuccess)
	}

	outputs := map[string]interface{}{
		"tx_count":                 t.config.TxCount,
		"avg_latency_mus":          avgLatency.Microseconds(),
		"tx_pool_latency_hdr_plot": plot,
		"latencies":                latenciesStats,
	}

	outputsJSON, _ := json.Marshal(outputs)
	t.logger.Infof("outputs_json: %s", string(outputsJSON))

	return nil
}

func (t *Task) getTcpConn(ctx context.Context, client *execution.Client) (*sentry.Conn, error) {
	chainConfig := params.AllDevChainProtocolChanges

	head, err := client.GetRPCClient().GetLatestBlock(ctx)
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

	t.logger.Infof("Connected to %s", client.GetName())

	return conn, nil
}

func (t *Task) generateTransaction(ctx context.Context) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		addr := t.wallet.GetAddress()
		toAddr := &addr

		txAmount, _ := crand.Int(crand.Reader, big.NewInt(0).SetUint64(10*1e18))

		feeCap := &helper.BigInt{Value: *big.NewInt(100000000000)} // 100 Gwei
		tipCap := &helper.BigInt{Value: *big.NewInt(1000000000)}   // 1 Gwei

		var txObj ethtypes.TxData

		txObj = &ethtypes.DynamicFeeTx{
			ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: &tipCap.Value,
			GasFeeCap: &feeCap.Value,
			Gas:       50000,
			To:        toAddr,
			Value:     txAmount,
			Data:      []byte{},
		}

		return ethtypes.NewTx(txObj), nil
	})

	if err != nil {
		return nil, err
	}

	return tx, nil
}
