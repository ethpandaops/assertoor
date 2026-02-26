package generateeoatransactions

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/ethpandaops/spamoor/txbuilder"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_eoa_transactions"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates normal eoa transactions and sends them to the network",
		Category:    "transaction",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	txIndex    uint64
	wallet     *spamoor.Wallet
	walletPool *spamoor.WalletPool

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

//nolint:gocyclo // ignore
func (t *Task) Execute(ctx context.Context) error {
	// load wallets
	privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
	if err != nil {
		return err
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	if t.config.ChildWallets == 0 {
		t.wallet, err = walletMgr.GetWalletByPrivkey(ctx, privKey)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet: %w", err)
		}

		t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", t.wallet.GetAddress().Hex(), t.wallet.GetNonce(), t.wallet.GetReadableBalance(18, 0, 4, false, false))
	} else {
		t.walletPool, err = walletMgr.GetWalletPoolByPrivkey(ctx, t.logger, privKey, &txmgr.WalletPoolConfig{
			WalletCount:   t.config.ChildWallets,
			WalletSeed:    t.config.WalletSeed,
			RefillAmount:  uint256.MustFromBig(t.config.RefillAmount),
			RefillBalance: uint256.MustFromBig(t.config.RefillMinBalance),
		})
		if err != nil {
			return fmt.Errorf("cannot initialize wallet pool: %w", err)
		}

		rootWallet := t.walletPool.GetRootWallet().GetWallet()
		t.logger.Infof("funding wallet: %v [nonce: %v]  %v ETH", rootWallet.GetAddress().Hex(), rootWallet.GetNonce(), rootWallet.GetReadableBalance(18, 0, 4, false, false))

		for idx, wallet := range t.walletPool.GetAllWallets() {
			t.logger.Infof("wallet #%v: %v [nonce: %v]  %v ETH", idx, wallet.GetAddress().Hex(), wallet.GetNonce(), wallet.GetReadableBalance(18, 0, 4, false, false))
		}
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

	pendingWaitGroup := sync.WaitGroup{}
	perBlockCount := 0
	totalCount := 0

	sucessCount := 0
	revertCount := 0
	unknownCount := 0

	t.ctx.ReportProgress(0, "Generating EOA transactions...")

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

		pendingWaitGroup.Add(1)

		err := t.generateTransaction(ctx, txIndex, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
			if pendingChan != nil {
				<-pendingChan
			}

			switch {
			case receipt != nil:
				t.logger.Infof("transaction %v confirmed in block %v (nonce: %v, status: %v)", tx.Hash().Hex(), receipt.BlockNumber, tx.Nonce(), receipt.Status)

				if receipt.Status == 0 {
					revertCount++
				} else {
					sucessCount++
				}
			case err != nil:
				t.logger.Errorf("error awaiting transaction receipt: %v", err.Error())

				unknownCount++
			default:
				t.logger.Warnf("no receipt for transaction: %v (maybe replaced?)", tx.Hash().Hex())

				unknownCount++
			}

			pendingWaitGroup.Done()
		})
		if err != nil {
			t.logger.Errorf("error generating transaction: %v", err.Error())

			// Note: onComplete callback is still called by spamoor even on error,
			// so we don't drain pendingChan or call pendingWaitGroup.Done() here
		} else {
			perBlockCount++
			totalCount++

			if t.config.LimitTotal > 0 {
				progress := float64(totalCount) / float64(t.config.LimitTotal) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d EOA transactions", totalCount, t.config.LimitTotal))
			} else {
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d EOA transactions", totalCount))
			}
		}

		if t.config.LimitTotal > 0 && totalCount >= t.config.LimitTotal {
			t.ctx.ReportProgress(100, fmt.Sprintf("Completed: generated %d EOA transactions", totalCount))

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

	if t.config.AwaitReceipt {
		pendingWaitGroup.Wait()
	}

	t.logger.Infof("seding complete, total sent: %v, success: %v, reverted: %v, unknown: %v, pending: %v", totalCount, sucessCount, revertCount, unknownCount, totalCount-(sucessCount+revertCount+unknownCount))

	switch {
	case t.config.FailOnSuccess && sucessCount > 0:
		t.logger.Infof("set task result to failed, %v transactions succeeded unexpectedly (FailOnSuccess)", sucessCount)
		t.ctx.SetResult(types.TaskResultFailure)
	case t.config.FailOnReject && revertCount > 0:
		t.logger.Infof("set task result to failed, %v transactions reverted unexpectedly (FailOnReject)", revertCount)
		t.ctx.SetResult(types.TaskResultFailure)
	case totalCount == 0:
		t.logger.Infof("set task result to failed, no transactions sent")
		t.ctx.SetResult(types.TaskResultFailure)
	}

	return nil
}

func (t *Task) generateTransaction(ctx context.Context, transactionIdx uint64, completeFn spamoor.TxCompleteFn) error {
	txWallet := t.wallet
	if t.wallet == nil {
		txWallet = t.walletPool.GetWallet(spamoor.SelectWalletRoundRobin, 0)
	}

	var toAddr *common.Address

	if !t.config.ContractDeployment {
		addr := txWallet.GetAddress()

		if t.config.RandomTarget {
			addrBytes := make([]byte, 20)
			//nolint:errcheck // ignore
			rand.Read(addrBytes)
			addr = common.Address(addrBytes)
		} else if t.config.TargetAddress != "" {
			addr = t.targetAddr
		}

		toAddr = &addr
	}

	txAmount := new(big.Int).Set(t.config.Amount)
	if t.config.RandomAmount {
		n, err := rand.Int(rand.Reader, txAmount)
		if err == nil {
			txAmount = n
		}
	}

	txData := []byte{}
	if t.transactionData != nil {
		txData = t.transactionData
	}

	var clients []*execution.Client

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().AwaitReadyEndpoints(ctx, true)
		if len(clients) == 0 {
			return ctx.Err()
		}
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

	var tx *ethtypes.Transaction

	var err error

	if t.config.LegacyTxType {
		legacyTx, buildErr := txbuilder.LegacyTx(&txbuilder.TxMetadata{
			GasFeeCap: uint256.MustFromBig(t.config.FeeCap),
			Gas:       t.config.GasLimit,
			To:        toAddr,
			Value:     uint256.MustFromBig(txAmount),
			Data:      txData,
		})
		if buildErr != nil {
			return fmt.Errorf("cannot build legacy tx data: %w", buildErr)
		}

		tx, err = txWallet.BuildLegacyTx(legacyTx)
	} else {
		dynFeeTx, buildErr := txbuilder.DynFeeTx(&txbuilder.TxMetadata{
			GasTipCap: uint256.MustFromBig(t.config.TipCap),
			GasFeeCap: uint256.MustFromBig(t.config.FeeCap),
			Gas:       t.config.GasLimit,
			To:        toAddr,
			Value:     uint256.MustFromBig(txAmount),
			Data:      txData,
		})
		if buildErr != nil {
			return fmt.Errorf("cannot build dynamic fee tx data: %w", buildErr)
		}

		tx, err = txWallet.BuildDynamicFeeTx(dynFeeTx)
	}

	if err != nil {
		return fmt.Errorf("cannot build transaction: %w", err)
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	spamoorClients := make([]*spamoor.Client, len(clients))
	for i, c := range clients {
		spamoorClients[i] = walletMgr.GetClient(c)
	}

	err = walletMgr.GetTxPool().SendTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      spamoorClients[transactionIdx%uint64(len(spamoorClients))],
		ClientList:  spamoorClients,
		Rebroadcast: true,
		OnComplete:  completeFn,
		LogFn: func(client *spamoor.Client, retry int, rebroadcast int, err error) {
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

			if retry > 0 {
				logEntry = logEntry.WithField("retry", retry)
			}

			logEntry.Infof("submitted tx %v: %v", transactionIdx, tx.Hash().Hex())
		},
	})
	if err != nil {
		txWallet.MarkSkippedNonce(tx.Nonce())
		return err
	}

	return nil
}
