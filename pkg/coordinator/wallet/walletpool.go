package wallet

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
)

//nolint:revive // ignore
type WalletPool struct {
	manager    *Manager
	logger     logrus.FieldLogger
	rootWallet *Wallet
	wallets    []*Wallet
	nextIdx    uint64
}

func (manager *Manager) GetWalletPoolByPrivkey(privkey *ecdsa.PrivateKey, walletCount uint64, childSeed string) (*WalletPool, error) {
	rootWallet, err := manager.GetWalletByPrivkey(privkey)
	if err != nil {
		return nil, err
	}

	pool := &WalletPool{
		manager:    manager,
		logger:     manager.logger,
		rootWallet: rootWallet,
		wallets:    make([]*Wallet, walletCount),
	}

	for i := uint64(0); i < walletCount; i++ {
		wallet, err := pool.newChildWallet(i, childSeed)
		if err != nil {
			return nil, err
		}

		pool.wallets[i] = wallet
	}

	return pool, nil
}

func (pool *WalletPool) newChildWallet(childIdx uint64, childSeed string) (*Wallet, error) {
	idxBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idxBytes, childIdx)

	if childSeed != "" {
		seedBytes := []byte(childSeed)
		idxBytes = append(idxBytes, seedBytes...)
	}

	childKeyBytes := sha256.Sum256(append(crypto.FromECDSA(pool.rootWallet.privkey), idxBytes...))

	childKey, err := crypto.ToECDSA(childKeyBytes[:])
	if err != nil {
		return nil, err
	}

	return pool.manager.GetWalletByPrivkey(childKey)
}

func (pool *WalletPool) GetRootWallet() *Wallet {
	return pool.rootWallet
}

func (pool *WalletPool) GetChildWallets() []*Wallet {
	return pool.wallets
}

func (pool *WalletPool) GetChildWallet(index uint64) *Wallet {
	if index >= uint64(len(pool.wallets)) {
		return nil
	}

	return pool.wallets[index]
}

func (pool *WalletPool) GetNextChildWallet() *Wallet {
	wallet := pool.wallets[pool.nextIdx]
	pool.nextIdx++

	if pool.nextIdx >= uint64(len(pool.wallets)) {
		pool.nextIdx = 0
	}

	return wallet
}

func (pool *WalletPool) EnsureFunding(ctx context.Context, minBalance, refillAmount, gasFeeCap, gasTipCap *big.Int, pendingLimit uint64) error {
	refillTxs := []*types.Transaction{}

	var refillError error

	for _, wallet := range pool.wallets {
		err := wallet.AwaitReady(ctx)
		if err != nil {
			return err
		}

		if wallet.GetPendingBalance().Cmp(minBalance) >= 1 {
			continue
		}

		amount := refillAmount
		if amount.Cmp(minBalance) < 0 {
			diffAmount := minBalance.Sub(minBalance, wallet.GetPendingBalance())
			if diffAmount.Cmp(amount) > 0 {
				amount = diffAmount
			}
		}

		tx, err := pool.rootWallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*types.Transaction, error) {
			toAddr := wallet.GetAddress()
			txData := &types.DynamicFeeTx{
				ChainID:   pool.manager.clientPool.GetBlockCache().GetChainID(),
				Nonce:     nonce,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Gas:       50000,
				To:        &toAddr,
				Value:     amount,
			}

			return types.NewTx(txData), nil
		})
		if err != nil {
			pool.logger.Warnf("failed creating child wallet refill tx: %v", err)
			refillError = err

			continue
		}

		refillTxs = append(refillTxs, tx)
	}

	refillTxCount := uint64(len(refillTxs))
	if refillTxCount == 0 {
		if refillError != nil {
			return refillError
		}

		return nil
	}

	sentIdx := uint64(0)
	headFork := pool.manager.clientPool.GetCanonicalFork(0)
	clients := headFork.ReadyClients
	txChan := make(chan bool, pendingLimit)
	txWg := sync.WaitGroup{}

	for sentIdx < refillTxCount {
		select {
		case txChan <- true:
		case <-ctx.Done():
			return ctx.Err()
		}

		tx := refillTxs[sentIdx]
		sentIdx++

		var err error

		for i := 0; i < len(clients); i++ {
			err = clients[i].GetRPCClient().SendTransaction(ctx, tx)
			if err == nil {
				break
			}
		}

		if err != nil {
			pool.logger.Warnf("failed sending child wallet refill tx: %v", err)
			refillError = err

			break
		}

		txWg.Add(1)

		go func(tx *types.Transaction) {
			_, err := pool.rootWallet.AwaitTransaction(ctx, tx)

			if err != nil {
				pool.logger.Warnf("failed awaiting child wallet refill tx: %v", err)
				refillError = err
			}

			<-txChan
			txWg.Done()
		}(tx)
	}

	txWg.Wait()

	return refillError
}
