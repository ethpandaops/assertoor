package generateblobtransactions

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet/blobtx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_blob_transactions"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates blob transactions and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	txIndex    uint64
	wallet     *wallet.Wallet
	walletPool *wallet.WalletPool

	targetAddr      common.Address
	transactionData []byte
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

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if valerr := config.Validate(); valerr != nil {
		return valerr
	}

	// load wallets
	privKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return err
	}

	if config.ChildWallets == 0 {
		t.wallet, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet: %w", err)
		}
	} else {
		t.walletPool, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletPoolByPrivkey(privKey, config.ChildWallets, config.WalletSeed)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet pool: %w", err)
		}
	}

	// parse target addr
	if config.TargetAddress != "" {
		err = t.targetAddr.UnmarshalText([]byte(config.TargetAddress))
		if err != nil {
			return fmt.Errorf("cannot decode execution addr: %w", err)
		}
	}

	// parse transaction data
	if config.CallData != "" {
		t.transactionData = common.FromHex(config.CallData)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.walletPool != nil {
		err := t.walletPool.GetRootWallet().AwaitReady(ctx)
		if err != nil {
			return err
		}

		t.logger.Infof("funding wallet: %v [nonce: %v]  %v ETH", t.walletPool.GetRootWallet().GetAddress().Hex(), t.walletPool.GetRootWallet().GetNonce(), t.walletPool.GetRootWallet().GetReadableBalance(18, 0, 4, false, false))

		err = t.ensureChildWalletFunding(ctx)
		if err != nil {
			t.logger.Infof("failed ensuring child wallet funding: %v", err)
			return err
		}

		for idx, wallet := range t.walletPool.GetChildWallets() {
			t.logger.Infof("wallet #%v: %v [nonce: %v]  %v ETH", idx, wallet.GetAddress().Hex(), wallet.GetNonce(), wallet.GetReadableBalance(18, 0, 4, false, false))
		}

		go t.runChildWalletFundingRoutine(ctx)
	} else {
		err := t.wallet.AwaitReady(ctx)
		if err != nil {
			return err
		}

		t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", t.wallet.GetAddress().Hex(), t.wallet.GetNonce(), t.wallet.GetReadableBalance(18, 0, 4, false, false))
	}

	var subscription *execution.Subscription[*execution.Block]
	if t.config.LimitPerBlock > 0 {
		subscription = t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	var pendingChan chan bool

	if t.config.LimitPending > 0 {
		pendingChan = make(chan bool, t.config.LimitPending)
	}

	perBlockCount := 0
	totalCount := 0

	for {
		txIndex := t.txIndex
		t.txIndex++

		if pendingChan != nil {
			select {
			case <-ctx.Done():
				return nil
			case pendingChan <- true:
			}
		}

		err := t.generateTransaction(ctx, txIndex, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
			if pendingChan != nil {
				<-pendingChan
			}

			switch {
			case receipt != nil:
				t.logger.Infof("blob %v confirmed in block %v (nonce: %v, status: %v)", tx.Hash().Hex(), receipt.BlockNumber, tx.Nonce(), receipt.Status)
			case err != nil:
				t.logger.Errorf("error awaiting blob transaction receipt: %v", err.Error())
			default:
				t.logger.Warnf("no receipt for blob transaction: %v", tx.Hash().Hex())
			}
		})
		if err != nil {
			t.logger.Errorf("error generating transaction: %v", err.Error())

			if pendingChan != nil {
				<-pendingChan
			}
		} else {
			perBlockCount++
			totalCount++
		}

		if t.config.LimitTotal > 0 && totalCount >= t.config.LimitTotal {
			break
		}

		if t.config.LimitPerBlock > 0 && perBlockCount >= t.config.LimitPerBlock {
			// await next block
			perBlockCount = 0
			select {
			case <-ctx.Done():
				return nil
			case <-subscription.Channel():
			}
		} else if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (t *Task) runChildWalletFundingRoutine(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Minute):
			err := t.ensureChildWalletFunding(ctx)
			if err != nil {
				t.logger.Infof("failed ensuring child wallet funding: %v", err)
			}
		}
	}
}

func (t *Task) ensureChildWalletFunding(ctx context.Context) error {
	t.logger.Infof("ensure child wallet funding")

	err := t.walletPool.EnsureFunding(ctx, t.config.RefillMinBalance, t.config.RefillAmount, t.config.RefillFeeCap, t.config.RefillTipCap, t.config.RefillPendingLimit)
	if err != nil {
		return err
	}

	return nil
}

func (t *Task) generateTransaction(ctx context.Context, transactionIdx uint64, confirmedFn wallet.TxConfirmFn) error {
	txWallet := t.wallet
	if t.wallet == nil {
		txWallet = t.walletPool.GetNextChildWallet()
	}

	blobRef := "identifier,random"
	if t.config.BlobData != "" {
		blobRef = t.config.BlobData
	}

	blobRefs := []string{}
	for i := uint64(0); i < t.config.BlobSidecars; i++ {
		blobRefs = append(blobRefs, blobRef)
	}

	blobHashes, blobSidecar, err := blobtx.GenerateBlobSidecar(blobRefs, transactionIdx, 0)
	if err != nil {
		return err
	}

	tx, err := txWallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		toAddr := txWallet.GetAddress()

		if t.config.RandomTarget {
			addrBytes := make([]byte, 20)
			//nolint:errcheck // ignore
			rand.Read(addrBytes)
			toAddr = common.Address(addrBytes)
		} else if t.config.TargetAddress != "" {
			toAddr = t.targetAddr
		}

		txAmount := new(big.Int).Set(t.config.Amount)
		if t.config.RandomAmount {
			n, err2 := rand.Int(rand.Reader, txAmount)
			if err2 == nil {
				txAmount = n
			}
		}

		txData := []byte{}
		if t.transactionData != nil {
			txData = t.transactionData
		}

		txObj := &ethtypes.BlobTx{
			ChainID:    uint256.MustFromBig(t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID()),
			Nonce:      nonce,
			BlobFeeCap: uint256.MustFromBig(t.config.BlobFeeCap),
			GasTipCap:  uint256.MustFromBig(t.config.TipCap),
			GasFeeCap:  uint256.MustFromBig(t.config.FeeCap),
			Gas:        t.config.GasLimit,
			To:         toAddr,
			Value:      uint256.MustFromBig(txAmount),
			Data:       txData,
			BlobHashes: blobHashes,
			Sidecar:    blobSidecar,
		}

		return ethtypes.NewTx(txObj), nil
	})
	if err != nil {
		return err
	}

	var clients []*execution.Client

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints(true)
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	if len(clients) == 0 {
		return fmt.Errorf("no ready clients available")
	}

	return txWallet.SendTransaction(ctx, tx, &wallet.SendTransactionOptions{
		Clients:            clients,
		ClientsStartOffset: transactionIdx,
		OnConfirm:          confirmedFn,
		LogFn: func(client *execution.Client, retry uint64, rebroadcast uint64, err error) {
			if err != nil {
				t.logger.WithFields(logrus.Fields{
					"client": client.GetName(),
				}).Warnf("error sending tx %v: %v", transactionIdx, err)
				return
			}

			logEntry := t.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
			})

			if rebroadcast > 0 {
				logEntry = logEntry.WithField("rebroadcast", rebroadcast)
			}

			logEntry.Infof("submitted blob transaction %v (nonce: %v, attempt: %v)", transactionIdx, tx.Nonce(), retry)
		},
		RebroadcastInterval: 30 * time.Second,
		MaxRebroadcasts:     5,
	})
}
