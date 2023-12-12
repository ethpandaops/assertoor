package execution

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
)

type Wallet struct {
	pool      *Pool
	address   common.Address
	privkey   *ecdsa.PrivateKey
	isReady   bool
	readyChan chan bool
	nonce     uint64
	balance   *big.Int
	txMutex   sync.Mutex

	nonceListener bool
	nonceChans    map[uint64]chan bool
	nonceMutex    sync.Mutex
}

type TxStatus struct {
}

func (pool *Pool) GetWalletByPrivkey(privkey *ecdsa.PrivateKey) (*Wallet, error) {
	publicKey := privkey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)

	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	wallet := pool.GetWalletByAddress(address)

	if wallet.privkey == nil {
		wallet.privkey = privkey
	}

	return wallet, nil
}

func (pool *Pool) GetWalletByAddress(address common.Address) *Wallet {
	pool.walletsMutex.Lock()
	defer pool.walletsMutex.Unlock()

	wallet := pool.walletsMap[address]
	if wallet == nil {
		wallet = newWallet(pool, address)
		pool.walletsMap[address] = wallet
	}

	return wallet
}

func newWallet(pool *Pool, address common.Address) *Wallet {
	wallet := &Wallet{
		pool:       pool,
		address:    address,
		nonceChans: map[uint64]chan bool{},
	}
	wallet.loadState()

	return wallet
}

func (wallet *Wallet) loadState() {
	wallet.readyChan = make(chan bool)

	go func() {
		for {
			client := wallet.pool.GetReadyEndpoint(UnspecifiedClient)
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

			wallet.nonce = nonce
			wallet.balance = balance
			wallet.isReady = true
			close(wallet.readyChan)
			cancel()

			break
		}
	}()
}

func (wallet *Wallet) GetAddress() common.Address {
	return wallet.address
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

	wallet.txMutex.Lock()
	defer wallet.txMutex.Unlock()

	signer := types.LatestSignerForChainID(wallet.pool.blockCache.GetChainID())
	nonce := wallet.nonce
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

	wallet.nonce++

	return signedTx, nil
}

func (wallet *Wallet) AwaitTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	nonceChan := wallet.getNonceIncreaseChan(tx.Nonce() + 1)
	if nonceChan != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-nonceChan:
		}
	}

	client := wallet.pool.GetCanonicalFork(0).ReadyClients[0]

	return client.GetRPCClient().GetTransactionReceipt(ctx, tx.Hash())
}

func (wallet *Wallet) getNonceIncreaseChan(targetNonce uint64) chan bool {
	wallet.nonceMutex.Lock()
	defer wallet.nonceMutex.Unlock()

	nonceChan := wallet.nonceChans[targetNonce]
	if nonceChan != nil {
		return nonceChan
	}

	nonceChan = make(chan bool)
	wallet.nonceChans[targetNonce] = nonceChan

	if !wallet.nonceListener {
		wallet.nonceListener = true

		go wallet.runNonceIncreaseLoop()
	}

	return nonceChan
}

func (wallet *Wallet) runNonceIncreaseLoop() {
	<-wallet.readyChan

	blockSubscription := wallet.pool.blockCache.blockDispatcher.Subscribe(10)
	defer blockSubscription.Unsubscribe()

	lastNonce := uint64(0)
	awaitNext := true

	for awaitNext {
		block := <-blockSubscription.Channel()
		client := block.GetSeenBy()[0]

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		nonce, err := client.GetRPCClient().GetNonceAt(ctx, wallet.address, nil)

		cancel()

		if err != nil {
			logrus.WithError(err).Warnf("could not get last noce for wallet %v", wallet.address.String())
			continue
		}

		if nonce == lastNonce {
			continue
		}

		wallet.nonceMutex.Lock()
		awaitNext = false
		lastNonce = nonce

		for n, c := range wallet.nonceChans {
			if n <= nonce {
				close(c)
				delete(wallet.nonceChans, n)
			} else {
				awaitNext = true
			}
		}

		if !awaitNext {
			wallet.nonceListener = false
		}

		wallet.nonceMutex.Unlock()
	}
}
