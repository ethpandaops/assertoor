package wallet

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"runtime/debug"
	"sync"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
)

type Manager struct {
	clientPool *execution.Pool
	logger     logrus.FieldLogger

	walletsMutex sync.Mutex
	walletsMap   map[common.Address]*Wallet
}

func NewManager(clientPool *execution.Pool, logger logrus.FieldLogger) *Manager {
	manager := &Manager{
		clientPool: clientPool,
		logger:     logger,
		walletsMap: map[common.Address]*Wallet{},
	}

	go manager.runBlockTransactionsLoop()

	return manager
}

func (manager *Manager) GetWalletByPrivkey(privkey *ecdsa.PrivateKey) (*Wallet, error) {
	publicKey := privkey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)

	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	wallet := manager.GetWalletByAddress(address)

	if wallet.privkey == nil {
		wallet.privkey = privkey
	}

	return wallet, nil
}

func (manager *Manager) GetWalletByAddress(address common.Address) *Wallet {
	manager.walletsMutex.Lock()
	defer manager.walletsMutex.Unlock()

	wallet := manager.walletsMap[address]
	if wallet == nil {
		wallet = manager.newWallet(address)
		manager.walletsMap[address] = wallet
	}

	return wallet
}

func (manager *Manager) runBlockTransactionsLoop() {
	defer func() {
		if err := recover(); err != nil {
			var err2 error
			if errval, errok := err.(error); errok {
				err2 = errval
			}

			manager.logger.WithError(err2).Panicf("uncaught panic in wallet.Manager.runBlockTransactionsLoop: %v, stack: %v", err, string(debug.Stack()))

			time.Sleep(10 * time.Second)

			go manager.runBlockTransactionsLoop()
		}
	}()

	blockSubscription := manager.clientPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	for block := range blockSubscription.Channel() {
		manager.processBlockTransactions(block)
	}
}

func (manager *Manager) processBlockTransactions(block *execution.Block) {
	blockData := block.AwaitBlock(context.Background(), 4*time.Second)
	if blockData == nil {
		return
	}

	manager.walletsMutex.Lock()

	wallets := map[common.Address]*Wallet{}

	for addr := range manager.walletsMap {
		wallets[addr] = manager.walletsMap[addr]
	}

	manager.walletsMutex.Unlock()

	manager.logger.Infof("processing block %v with %v transactions", block.Number, len(blockData.Transactions()))

	var blockReceipts []*ethtypes.Receipt

	receiptsLoaded := false

	signer := ethtypes.LatestSignerForChainID(manager.clientPool.GetBlockCache().GetChainID())

	for idx, tx := range blockData.Transactions() {
		txFrom, err := ethtypes.Sender(signer, tx)
		if err != nil {
			manager.logger.Warnf("error decoding tx sender (block %v, tx %v): %v", block.Number, idx, err)
			continue
		}

		fromWallet := wallets[txFrom]
		if fromWallet != nil {
			if !receiptsLoaded {
				blockReceipts = manager.loadBlockReceipts(block)
				receiptsLoaded = true
			}

			var txReceipt *ethtypes.Receipt

			if blockReceipts != nil && idx < len(blockReceipts) {
				txReceipt = blockReceipts[idx]
			}

			fromWallet.processTransactionInclusion(block, tx, txReceipt)
		}

		toAddr := tx.To()
		if toAddr != nil {
			toWallet := wallets[*toAddr]
			if toWallet != nil {
				toWallet.processTransactionReceival(block, tx)
			}
		}

		if tx.Type() == ethtypes.SetCodeTxType {
			// in eip7702 transactions, the nonces of all authorities are increased by >= 1, so we need to resync all affected wallets
			authorizations := tx.SetCodeAuthorizations()
			for i := 0; i < len(authorizations); i++ {
				authority, err := authorizations[i].Authority()
				if err != nil {
					manager.logger.Warnf("error decoding authority address (block %v, tx %v): %v", block.Number, idx, err)
					continue
				}

				authorityWallet := wallets[authority]
				if authorityWallet != nil {
					authorityWallet.ResyncState()
				}
			}
		}
	}

	for _, wallet := range wallets {
		wallet.processStaleConfirmations(block)
	}
}

func (manager *Manager) loadBlockReceipts(block *execution.Block) []*ethtypes.Receipt {
	retryCount := uint64(0)
	readyClients := manager.clientPool.GetReadyEndpoints(true)

	for {
		clients := block.GetSeenBy()
		if len(clients) == 0 {
			clients = readyClients
		}

		cliIdx := retryCount % uint64(len(clients))
		client := clients[cliIdx]

		receipts, err := manager.loadBlockReceiptsFromClient(client, block)
		if err == nil {
			return receipts
		}

		if retryCount > 2 {
			manager.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
				"block":  block.Number,
				"hash":   block.Hash.Hex(),
			}).Warnf("could not load block receipts: %v", err)
		}

		if retryCount < 5 {
			time.Sleep(1 * time.Second)

			retryCount++
		} else {
			return nil
		}
	}
}

func (manager *Manager) loadBlockReceiptsFromClient(client *execution.Client, block *execution.Block) ([]*ethtypes.Receipt, error) {
	reqCtx, reqCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer reqCtxCancel()

	return client.GetRPCClient().GetBlockReceipts(reqCtx, block.Hash)
}
