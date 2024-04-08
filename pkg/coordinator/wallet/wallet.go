package wallet

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

type Wallet struct {
	manager *Manager

	address      common.Address
	privkey      *ecdsa.PrivateKey
	isReady      bool
	isSyncing    bool
	readyChan    chan bool
	syncingMutex sync.Mutex

	txBuildMutex   sync.Mutex
	pendingNonce   uint64
	pendingBalance *big.Int

	confirmedNonce   uint64
	confirmedBalance *big.Int
	lastConfirmation uint64

	txNonceChans map[uint64]*nonceStatus
	txNonceMutex sync.Mutex
}

type nonceStatus struct {
	receipt *types.Receipt
	channel chan bool
}

func (manager *Manager) newWallet(address common.Address) *Wallet {
	wallet := &Wallet{
		manager:      manager,
		address:      address,
		txNonceChans: map[uint64]*nonceStatus{},
	}
	wallet.loadState()

	return wallet
}

func (wallet *Wallet) loadState() {
	wallet.syncingMutex.Lock()
	alreadySyncing := false

	if wallet.isSyncing {
		alreadySyncing = true
	} else {
		wallet.isSyncing = true
	}
	wallet.syncingMutex.Unlock()

	if alreadySyncing {
		return
	}

	wallet.readyChan = make(chan bool)

	go func() {
		for {
			client := wallet.manager.clientPool.GetReadyEndpoint(execution.AnyClient)
			if client == nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

			nonce, err := client.GetRPCClient().GetNonceAt(ctx, wallet.address, nil)
			if err != nil {
				logrus.WithError(err).Warnf("could not get last noce for wallet %v", wallet.address.String())
				cancel()

				continue
			}

			balance, err := client.GetRPCClient().GetBalanceAt(ctx, wallet.address, nil)
			if err != nil {
				logrus.WithError(err).Warnf("could not get balance for wallet %v", wallet.address.String())
				cancel()

				continue
			}

			wallet.pendingNonce = nonce
			wallet.confirmedNonce = nonce
			wallet.pendingBalance = new(big.Int).Set(balance)
			wallet.confirmedBalance = new(big.Int).Set(balance)
			wallet.isReady = true
			wallet.isSyncing = false
			close(wallet.readyChan)
			cancel()

			break
		}
	}()
}

func (wallet *Wallet) ResyncState() {
	wallet.loadState()
}

func (wallet *Wallet) GetAddress() common.Address {
	return wallet.address
}

func (wallet *Wallet) GetPrivateKey() *ecdsa.PrivateKey {
	return wallet.privkey
}

func (wallet *Wallet) GetBalance() *big.Int {
	return wallet.confirmedBalance
}

func (wallet *Wallet) GetPendingBalance() *big.Int {
	return wallet.pendingBalance
}

func (wallet *Wallet) GetNonce() uint64 {
	return wallet.confirmedNonce
}

func (wallet *Wallet) GetReadableBalance(unitDigits, maxPreCommaDigitsBeforeTrim, digits int, addPositiveSign, trimAmount bool) string {
	amount := wallet.GetPendingBalance()
	return GetReadableBalance(amount, unitDigits, maxPreCommaDigitsBeforeTrim, digits, addPositiveSign, trimAmount)
}

func (wallet *Wallet) AwaitReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wallet.readyChan:
	}

	return nil
}

func (wallet *Wallet) BuildTransaction(ctx context.Context, buildFn func(ctx context.Context, nonce uint64, signer bind.SignerFn) (*types.Transaction, error)) (*types.Transaction, error) {
	err := wallet.AwaitReady(ctx)
	if err != nil {
		return nil, err
	}

	wallet.txBuildMutex.Lock()
	defer wallet.txBuildMutex.Unlock()

	signer := types.LatestSignerForChainID(wallet.manager.clientPool.GetBlockCache().GetChainID())
	nonce := wallet.pendingNonce
	tx, err := buildFn(ctx, nonce, func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if !bytes.Equal(addr[:], wallet.address[:]) {
			return nil, fmt.Errorf("cannot sign for another wallet")
		}

		signedTx, serr := types.SignTx(tx, signer, wallet.privkey)
		if serr != nil {
			return nil, serr
		}

		return signedTx, nil
	})

	if err != nil {
		return nil, err
	}

	signedTx, err := types.SignTx(tx, signer, wallet.privkey)

	if err != nil {
		return nil, err
	}

	wallet.pendingNonce++
	wallet.pendingBalance = wallet.pendingBalance.Sub(wallet.pendingBalance, tx.Value())

	return signedTx, nil
}

func (wallet *Wallet) AwaitTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	err := wallet.AwaitReady(ctx)
	if err != nil {
		return nil, err
	}

	txHash := tx.Hash()
	nonceChan := wallet.getTxNonceChan(tx.Nonce())

	if nonceChan != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-nonceChan.channel:
		}

		receipt := nonceChan.receipt
		if receipt != nil {
			if bytes.Equal(receipt.TxHash[:], txHash[:]) {
				return receipt, nil
			}

			return nil, nil
		}
	}

	client := wallet.manager.clientPool.AwaitReadyEndpoint(ctx, execution.AnyClient)
	if client == nil {
		return nil, ctx.Err()
	}

	return client.GetRPCClient().GetTransactionReceipt(ctx, txHash)
}

func (wallet *Wallet) getTxNonceChan(targetNonce uint64) *nonceStatus {
	wallet.txNonceMutex.Lock()
	defer wallet.txNonceMutex.Unlock()

	nonceChan := wallet.txNonceChans[targetNonce]
	if nonceChan != nil {
		return nonceChan
	}

	nonceChan = &nonceStatus{
		channel: make(chan bool),
	}
	wallet.txNonceChans[targetNonce] = nonceChan

	return nonceChan
}

func (wallet *Wallet) processTransactionInclusion(block *execution.Block, tx *types.Transaction) {
	if !wallet.isReady {
		return
	}

	receipt := wallet.loadTransactionReceipt(block, tx)
	nonce := tx.Nonce() + 1

	wallet.txNonceMutex.Lock()
	defer wallet.txNonceMutex.Unlock()

	if wallet.confirmedNonce >= nonce {
		return
	}

	wallet.lastConfirmation = block.Number

	if receipt != nil {
		wallet.confirmedBalance = wallet.confirmedBalance.Sub(wallet.confirmedBalance, tx.Value())
		txFee := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(int64(receipt.GasUsed)))
		wallet.confirmedBalance = wallet.confirmedBalance.Sub(wallet.confirmedBalance, txFee)
		wallet.pendingBalance = wallet.pendingBalance.Sub(wallet.pendingBalance, txFee)
	}

	for n := range wallet.txNonceChans {
		if n == nonce-1 {
			wallet.txNonceChans[n].receipt = receipt
		}

		if n < nonce {
			close(wallet.txNonceChans[n].channel)
			delete(wallet.txNonceChans, n)
		}
	}

	wallet.confirmedNonce = nonce
	if wallet.confirmedNonce > wallet.pendingNonce {
		wallet.pendingNonce = wallet.confirmedNonce
		wallet.pendingBalance = new(big.Int).Set(wallet.confirmedBalance)
	}
}

func (wallet *Wallet) processStaleConfirmations(block *execution.Block) {
	if !wallet.isReady {
		return
	}

	if len(wallet.txNonceChans) > 0 && block.Number > wallet.lastConfirmation+10 {
		wallet.lastConfirmation = block.Number
		clients := block.GetSeenBy()
		client := clients[0]

		lastNonce, err := client.GetRPCClient().GetNonceAt(context.Background(), wallet.address, big.NewInt(int64(block.Number)))
		if err != nil {
			return
		}

		wallet.txNonceMutex.Lock()
		defer wallet.txNonceMutex.Unlock()

		if wallet.confirmedNonce >= lastNonce {
			return
		}

		for n := range wallet.txNonceChans {
			if n < lastNonce {
				logrus.WithError(err).Warnf("recovering stale confirmed transactions for %v (nonce %v)", wallet.address.String(), n)
				close(wallet.txNonceChans[n].channel)
				delete(wallet.txNonceChans, n)
			}
		}
	}
}

func (wallet *Wallet) processTransactionReceival(_ *execution.Block, tx *types.Transaction) {
	if !wallet.isReady {
		return
	}

	wallet.pendingBalance = wallet.pendingBalance.Add(wallet.pendingBalance, tx.Value())
	wallet.confirmedBalance = wallet.confirmedBalance.Add(wallet.confirmedBalance, tx.Value())
}

func (wallet *Wallet) loadTransactionReceipt(block *execution.Block, tx *types.Transaction) *types.Receipt {
	retryCount := uint64(0)

	for {
		clients := block.GetSeenBy()
		cliIdx := retryCount % uint64(len(clients))
		client := clients[cliIdx]

		receipt, err := client.GetRPCClient().GetTransactionReceipt(context.Background(), tx.Hash())
		if err == nil {
			return receipt
		}

		if retryCount > 2 {
			wallet.manager.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
				"txhash": tx.Hash(),
			}).Warnf("could not load tx receipt: %v", err)
		}

		if retryCount < 5 {
			time.Sleep(1 * time.Second)

			retryCount++
		} else {
			return nil
		}
	}
}
