package wallet

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
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
	blockSubscription := manager.clientPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	for block := range blockSubscription.Channel() {
		manager.processBlockTransactions(block)
	}
}

func (manager *Manager) processBlockTransactions(block *execution.Block) {
	blockData := block.AwaitBlock(context.Background(), 2*time.Second)
	if blockData == nil {
		return
	}

	manager.walletsMutex.Lock()

	wallets := map[common.Address]*Wallet{}
	for addr := range manager.walletsMap {
		wallets[addr] = manager.walletsMap[addr]
	}

	manager.walletsMutex.Unlock()

	signer := ethtypes.LatestSignerForChainID(manager.clientPool.GetBlockCache().GetChainID())
	for idx, tx := range blockData.Transactions() {
		txFrom, err := ethtypes.Sender(signer, tx)
		if err != nil {
			manager.logger.Warnf("error decoding ts sender (block %v, tx %v): %v", block.Number, idx, err)
			continue
		}

		fromWallet := wallets[txFrom]
		if fromWallet != nil {
			fromWallet.processTransactionInclusion(block, tx)
		}

		toAddr := tx.To()
		if toAddr != nil {
			toWallet := wallets[*toAddr]
			if toWallet != nil {
				toWallet.processTransactionReceival(block, tx)
			}
		}
	}
}
