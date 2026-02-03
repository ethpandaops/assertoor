package txmgr

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

type Spamoor struct {
	executionPool *execution.Pool
	clientPool    *spamoor.ClientPool
	txpool        *spamoor.TxPool
	clients       map[*execution.Client]*spamoor.Client
}

func NewSpamoor(ctx context.Context, logger logrus.FieldLogger, executionPool *execution.Pool) (*Spamoor, error) {
	s := &Spamoor{
		executionPool: executionPool,
		clients:       make(map[*execution.Client]*spamoor.Client),
	}

	clientPool := spamoor.NewClientPool(ctx, logger.WithField("module", "clientpool"))
	clientOptions := make([]*spamoor.ClientOptions, 0)

	endpoints := executionPool.GetAllEndpoints()
	for _, client := range endpoints {
		clientOptions = append(clientOptions, s.getClientOptions(client))
	}

	err := clientPool.InitClients(clientOptions)
	if err != nil {
		return nil, err
	}

	for i, client := range clientPool.GetAllClients() {
		s.clients[endpoints[i]] = client
	}

	txpool := spamoor.NewTxPool(&spamoor.TxPoolOptions{
		Context:    ctx,
		Logger:     logger.WithField("module", "txpool"),
		ClientPool: clientPool,
		ChainId:    clientPool.GetChainId(),

		ExternalBlockSource: &spamoor.ExternalBlockSource{
			SubscribeBlocks: func(ctx context.Context, capacity int) chan *spamoor.ExternalBlockEvent {
				blockSubscription := executionPool.GetBlockCache().SubscribeBlockEvent(10)
				blockEventChan := make(chan *spamoor.ExternalBlockEvent, capacity)

				go s.forwardBlocks(ctx, blockSubscription, blockEventChan)

				return blockEventChan
			},
		},
	})

	err = clientPool.PrepareClients()
	if err != nil {
		return nil, err
	}

	s.clientPool = clientPool
	s.txpool = txpool

	return s, nil
}

func (s *Spamoor) getClientOptions(client *execution.Client) *spamoor.ClientOptions {
	rpcURL := client.GetEndpointConfig().URL
	rpcURL = fmt.Sprintf("name(%s)%s", client.GetName(), rpcURL)

	if headers := client.GetEndpointConfig().Headers; len(headers) > 0 {
		for key, value := range headers {
			rpcURL = fmt.Sprintf("header(%s: %s)%s", key, value, rpcURL)
		}
	}

	opts := &spamoor.ClientOptions{
		RpcHost: rpcURL,
		ExternalClient: &spamoor.ExternalClientOptions{
			GetBlockHeight: func(_ context.Context) (uint64, error) {
				blockNum, _ := client.GetLastHead()
				return blockNum, nil
			},
		},
	}

	return opts
}

func (s *Spamoor) forwardBlocks(ctx context.Context, blockSubscription *execution.Subscription[*execution.Block], blockEventChan chan *spamoor.ExternalBlockEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case block := <-blockSubscription.Channel():
			time.Sleep(1 * time.Second) // wait for block to be seen by all clients

			seenBy := block.GetSeenBy()

			spamoorClients := make([]*spamoor.Client, 0, len(seenBy))

			for _, client := range seenBy {
				spamoorClient := s.GetClient(client)
				if spamoorClient == nil {
					continue
				}

				spamoorClients = append(spamoorClients, spamoorClient)
			}

			select {
			case <-ctx.Done():
				return
			case blockEventChan <- &spamoor.ExternalBlockEvent{
				Number:  block.Number,
				Clients: spamoorClients,
				Block: &spamoor.BlockWithHash{
					Hash:  block.Hash,
					Block: block.AwaitBlock(ctx, 10*time.Second),
				},
			}:
			}
		}
	}
}

func (s *Spamoor) GetClientPool() *spamoor.ClientPool {
	return s.clientPool
}

func (s *Spamoor) GetTxPool() *spamoor.TxPool {
	return s.txpool
}

func (s *Spamoor) GetReadyClient() *spamoor.Client {
	return s.clientPool.GetClient()
}

func (s *Spamoor) GetClient(client *execution.Client) *spamoor.Client {
	return s.clients[client]
}

func (s *Spamoor) GetWalletByPrivkey(ctx context.Context, privkey *ecdsa.PrivateKey) (*spamoor.Wallet, error) {
	publicKey := privkey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)

	if !ok {
		return nil, errors.New("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	wallet := spamoor.NewWallet(privkey, address)
	wallet = s.txpool.RegisterWallet(wallet, ctx)

	client := s.GetReadyClient()

	err := wallet.UpdateWallet(ctx, client, false)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

func (s *Spamoor) GetWalletByAddress(ctx context.Context, address common.Address) (*spamoor.Wallet, error) {
	wallet := spamoor.NewWallet(nil, address)
	wallet = s.txpool.RegisterWallet(wallet, ctx)

	client := s.GetReadyClient()

	err := wallet.UpdateWallet(ctx, client, false)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}

// WalletPoolConfig holds configuration for creating a wallet pool.
type WalletPoolConfig struct {
	WalletCount   uint64
	WalletSeed    string
	RefillAmount  *uint256.Int
	RefillBalance *uint256.Int
}

// GetWalletPoolByPrivkey creates a wallet pool with child wallets derived from the given private key.
// This method prepares the wallets immediately after creation.
func (s *Spamoor) GetWalletPoolByPrivkey(ctx context.Context, logger logrus.FieldLogger, privkey *ecdsa.PrivateKey, config *WalletPoolConfig) (*spamoor.WalletPool, error) {
	walletPool, err := s.NewWalletPoolByPrivkey(ctx, logger, privkey)
	if err != nil {
		return nil, err
	}

	walletPool.SetWalletCount(config.WalletCount)
	walletPool.SetWalletSeed(config.WalletSeed)

	if config.RefillAmount != nil {
		walletPool.SetRefillAmount(config.RefillAmount)
	}

	if config.RefillBalance != nil {
		walletPool.SetRefillBalance(config.RefillBalance)
	}

	err = walletPool.PrepareWallets()
	if err != nil {
		return nil, fmt.Errorf("cannot prepare wallets: %w", err)
	}

	return walletPool, nil
}

// NewWalletPoolByPrivkey creates a wallet pool without preparation.
// Use this when the wallet count and config will be set by a scenario.
// Call walletPool.PrepareWallets() after configuration is complete.
func (s *Spamoor) NewWalletPoolByPrivkey(ctx context.Context, logger logrus.FieldLogger, privkey *ecdsa.PrivateKey) (*spamoor.WalletPool, error) {
	privkeyHex := fmt.Sprintf("%x", crypto.FromECDSA(privkey))

	rootWallet, err := spamoor.InitRootWallet(ctx, privkeyHex, s.clientPool, s.txpool, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize root wallet: %w", err)
	}

	walletPool := spamoor.NewWalletPool(ctx, logger, rootWallet, s.clientPool, s.txpool)

	return walletPool, nil
}
