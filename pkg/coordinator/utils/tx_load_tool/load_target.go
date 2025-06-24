package txloadtool

import (
	"context"
	crand "crypto/rand"
	"fmt"
	"math/big"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/erigontech/assertoor/pkg/coordinator/helper"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/sirupsen/logrus"
)

// LoadTarget represents the target for the load test
type LoadTarget struct {
	ctx     context.Context
	taskCtx *types.TaskContext
	wallet  *wallet.Wallet
	logger  logrus.FieldLogger
	node    *execution.Client
}

// NewLoadTarget creates a new LoadTarget instance
func NewLoadTarget(ctx context.Context, taskCtx *types.TaskContext, logger logrus.FieldLogger,
	w *wallet.Wallet, client *execution.Client) *LoadTarget {
	return &LoadTarget{
		ctx:     ctx,
		taskCtx: taskCtx,
		wallet:  w,
		logger:  logger,
		node:    client,
	}
}

// GenerateTransaction creates a new transaction for the load test
func (t *LoadTarget) GenerateTransaction(i int) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(t.ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		addr := t.wallet.GetAddress()
		toAddr := &addr

		txAmount, _ := crand.Int(crand.Reader, big.NewInt(0).SetUint64(10*1e18))

		feeCap := &helper.BigInt{Value: *big.NewInt(100000000000)} // 100 Gwei
		tipCap := &helper.BigInt{Value: *big.NewInt(1000000000)}   // 1 Gwei

		txObj := &ethtypes.DynamicFeeTx{
			ChainID:   t.taskCtx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: &tipCap.Value,
			GasFeeCap: &feeCap.Value,
			Gas:       50000,
			To:        toAddr,
			Value:     txAmount,
			Data:      []byte(fmt.Sprintf("txIndex:%d", i)),
		}

		return ethtypes.NewTx(txObj), nil
	})

	if err != nil {
		return nil, err
	}

	return tx, nil
}

// SendTransaction sends a transaction to the execution node
func (t *LoadTarget) SendTransaction(tx *ethtypes.Transaction) error {
	return t.node.GetRPCClient().SendTransaction(t.ctx, tx)
}
