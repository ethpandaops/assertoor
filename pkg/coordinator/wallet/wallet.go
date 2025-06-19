package wallet

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

type Summary struct {
	Address          string   `json:"address" yaml:"address"`
	PrivKey          string   `json:"privkey" yaml:"privkey"`
	PendingBalance   *big.Int `json:"pendingBalance" yaml:"pendingBalance"`
	PendingNonce     uint64   `json:"pendingNonce" yaml:"pendingNonce"`
	ConfirmedBalance *big.Int `json:"confirmedBalance" yaml:"confirmedBalance"`
	ConfirmedNonce   uint64   `json:"confirmedNonce" yaml:"confirmedNonce"`
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

func (wallet *Wallet) GetSummary() *Summary {
	return &Summary{
		Address:          wallet.address.String(),
		PrivKey:          fmt.Sprintf("%x", crypto.FromECDSA(wallet.privkey)),
		PendingBalance:   wallet.pendingBalance,
		PendingNonce:     wallet.pendingNonce,
		ConfirmedBalance: wallet.confirmedBalance,
		ConfirmedNonce:   wallet.confirmedNonce,
	}
}

func (wallet *Wallet) GetReadableBalance(unitDigits, maxPreCommaDigitsBeforeTrim, digits int, addPositiveSign, trimAmount bool) string {
	// Initialize trimmedAmount and postComma variables to "0"
	fullAmount := ""
	trimmedAmount := "0"
	postComma := "0"
	proceed := ""
	amount := wallet.GetPendingBalance()

	if amount != nil {
		s := amount.String()

		if amount.Sign() > 0 && addPositiveSign {
			proceed = "+"
		} else if amount.Sign() < 0 {
			proceed = "-"
			s = strings.Replace(s, "-", "", 1)
		}

		l := len(s)

		// Check if there is a part of the amount before the decimal point
		switch {
		case l > unitDigits:
			// Calculate length of preComma part
			l -= unitDigits
			// Set preComma to part of the string before the decimal point
			trimmedAmount = s[:l]
			// Set postComma to part of the string after the decimal point, after removing trailing zeros
			postComma = strings.TrimRight(s[l:], "0")

			// Check if the preComma part exceeds the maximum number of digits before the decimal point
			if maxPreCommaDigitsBeforeTrim > 0 && l > maxPreCommaDigitsBeforeTrim {
				// Reduce the number of digits after the decimal point by the excess number of digits in the preComma part
				l -= maxPreCommaDigitsBeforeTrim
				if digits < l {
					digits = 0
				} else {
					digits -= l
				}
			}
			// Check if there is only a part of the amount after the decimal point, and no leading zeros need to be added
		case l == unitDigits:
			// Set postComma to part of the string after the decimal point, after removing trailing zeros
			postComma = strings.TrimRight(s, "0")
			// Check if there is only a part of the amount after the decimal point, and leading zeros need to be added
		case l != 0:
			// Use fmt package to add leading zeros to the string
			d := fmt.Sprintf("%%0%dd", unitDigits-l)
			// Set postComma to resulting string, after removing trailing zeros
			postComma = strings.TrimRight(fmt.Sprintf(d, 0)+s, "0")
		}

		fullAmount = trimmedAmount
		if postComma != "" {
			fullAmount += "." + postComma
		}

		// limit floating part
		if len(postComma) > digits {
			postComma = postComma[:digits]
		}

		// set floating point
		if postComma != "" {
			trimmedAmount += "." + postComma
		}
	}

	if trimAmount {
		return proceed + trimmedAmount
	}

	return proceed + fullAmount
}

func (wallet *Wallet) AwaitReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wallet.readyChan:
	}

	return nil
}

func (wallet *Wallet) UseNextNonce(increase bool) uint64 {
	nextNonce := wallet.pendingNonce

	if increase {
		wallet.pendingNonce++
	}

	return nextNonce
}

func (wallet *Wallet) BuildTransaction(ctx context.Context, buildFn func(ctx context.Context, nonce uint64, signer bind.SignerFn) (*types.Transaction, error)) (*types.Transaction, error) {
	err := wallet.AwaitReady(ctx)
	if err != nil {
		return nil, err
	}

	wallet.txBuildMutex.Lock()
	defer wallet.txBuildMutex.Unlock()

	signer := types.NewPragueSigner(wallet.manager.clientPool.GetBlockCache().GetChainID())
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

	if wallet.pendingNonce < tx.Nonce()+1 {
		wallet.pendingNonce = tx.Nonce() + 1
	}

	wallet.pendingBalance = wallet.pendingBalance.Sub(wallet.pendingBalance, tx.Value())

	return signedTx, nil
}

type TxConfirmFn func(tx *types.Transaction, receipt *types.Receipt, err error)
type TxLogFn func(client *execution.Client, retry uint64, rebroadcast uint64, err error)

type SendTransactionOptions struct {
	Clients            []*execution.Client
	ClientsStartOffset uint64

	OnConfirm TxConfirmFn
	LogFn     TxLogFn

	MaxRebroadcasts     uint64
	RebroadcastInterval time.Duration
}

func (wallet *Wallet) SendTransaction(ctx context.Context, tx *types.Transaction, options *SendTransactionOptions) error {
	var confirmCtx context.Context

	var confirmCancel context.CancelFunc

	if options.OnConfirm != nil || options.MaxRebroadcasts > 0 {
		confirmCtx, confirmCancel = context.WithCancel(ctx)
		wg := &sync.WaitGroup{}
		wg.Add(1)

		go func() {
			var receipt *types.Receipt

			var err error

			defer confirmCancel()

			if options.OnConfirm != nil {
				defer func() {
					options.OnConfirm(tx, receipt, err)
				}()
			}

			receipt, err = wallet.awaitTransaction(confirmCtx, tx, wg)
			if confirmCtx.Err() != nil {
				err = nil
			}
		}()

		wg.Wait()
	}

	var err error

	for i := uint64(0); i < uint64(len(options.Clients)); i++ {
		client := options.Clients[(i+options.ClientsStartOffset)%uint64(len(options.Clients))]

		err = client.GetRPCClient().SendTransaction(ctx, tx)

		if options.LogFn != nil {
			options.LogFn(client, i, 0, err)
		}

		if err == nil {
			break
		}
	}

	if err != nil {
		if confirmCancel != nil {
			confirmCancel()
		}

		return err
	}

	if options.MaxRebroadcasts > 0 {
		go func() {
			for i := uint64(0); i < options.MaxRebroadcasts; i++ {
				select {
				case <-confirmCtx.Done():
					return
				case <-time.After(options.RebroadcastInterval):
				}

				for j := uint64(0); j < uint64(len(options.Clients)); j++ {
					client := options.Clients[(i+j+options.ClientsStartOffset+1)%uint64(len(options.Clients))]

					err = client.GetRPCClient().SendTransaction(ctx, tx)

					if options.LogFn != nil {
						options.LogFn(client, j, i+1, err)
					}

					if err == nil {
						break
					}
				}
			}
		}()
	}

	return nil
}

func (wallet *Wallet) awaitTransaction(ctx context.Context, tx *types.Transaction, wg *sync.WaitGroup) (*types.Receipt, error) {
	err := wallet.AwaitReady(ctx)
	if err != nil {
		return nil, err
	}

	txHash := tx.Hash()
	nonceChan := wallet.getTxNonceChan(tx.Nonce())

	if wg != nil {
		wg.Done()
	}

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

	return wallet.loadTransactionReceipt(ctx, nil, tx), nil
}

func (wallet *Wallet) AwaitTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	return wallet.awaitTransaction(ctx, tx, nil)
}

func (wallet *Wallet) getTxNonceChan(targetNonce uint64) *nonceStatus {
	wallet.txNonceMutex.Lock()
	defer wallet.txNonceMutex.Unlock()

	if wallet.confirmedNonce > targetNonce {
		return nil
	}

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

func (wallet *Wallet) processTransactionInclusion(block *execution.Block, tx *types.Transaction, receipt *types.Receipt) {
	if !wallet.isReady {
		return
	}

	nonce := tx.Nonce()

	wallet.txNonceMutex.Lock()
	defer wallet.txNonceMutex.Unlock()

	if wallet.confirmedNonce > nonce {
		return
	}

	wallet.lastConfirmation = block.Number

	if receipt != nil {
		wallet.confirmedBalance = wallet.confirmedBalance.Sub(wallet.confirmedBalance, tx.Value())
		txFee := new(big.Int).Mul(receipt.EffectiveGasPrice, big.NewInt(0).SetUint64(receipt.GasUsed))
		wallet.confirmedBalance = wallet.confirmedBalance.Sub(wallet.confirmedBalance, txFee)
		wallet.pendingBalance = wallet.pendingBalance.Sub(wallet.pendingBalance, txFee)
	}

	for n := range wallet.txNonceChans {
		if n == nonce {
			wallet.txNonceChans[n].receipt = receipt
		}

		if n <= nonce {
			close(wallet.txNonceChans[n].channel)
			delete(wallet.txNonceChans, n)
		}
	}

	wallet.confirmedNonce = nonce + 1
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

		lastNonce, err := client.GetRPCClient().GetNonceAt(context.Background(), wallet.address, big.NewInt(0).SetUint64(block.Number))
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

func (wallet *Wallet) loadTransactionReceipt(ctx context.Context, block *execution.Block, tx *types.Transaction) *types.Receipt {
	retryCount := uint64(0)
	readyClients := wallet.manager.clientPool.GetReadyEndpoints(true)

	var clients []*execution.Client

	for {
		if block != nil {
			clients = block.GetSeenBy()
		}

		if len(clients) == 0 {
			clients = readyClients
		}

		cliIdx := retryCount % uint64(len(clients))
		client := clients[cliIdx]

		receipt, err := wallet.loadTransactionReceiptFromClient(ctx, client, tx)
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

func (wallet *Wallet) loadTransactionReceiptFromClient(ctx context.Context, client *execution.Client, tx *types.Transaction) (*types.Receipt, error) {
	reqCtx, reqCtxCancel := context.WithTimeout(ctx, 5*time.Second)
	defer reqCtxCancel()

	return client.GetRPCClient().GetTransactionReceipt(reqCtx, tx.Hash())
}
