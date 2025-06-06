package generateeoatransactions

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_eoa_transactions"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates normal eoa transactions and sends them to the network",
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

//nolint:gocyclo // ignore
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

	pendingWaitGroup := sync.WaitGroup{}
	perBlockCount := 0
	totalCount := 0

	sucessCount := 0
	revertCount := 0
	unknownCount := 0

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

			if pendingChan != nil {
				<-pendingChan
			}

			pendingWaitGroup.Done()
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

	tx, err := txWallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
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

		var txObj ethtypes.TxData

		if t.config.LegacyTxType {
			txObj = &ethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: t.config.FeeCap,
				Gas:      t.config.GasLimit,
				To:       toAddr,
				Value:    txAmount,
				Data:     txData,
			}
		} else {
			txObj = &ethtypes.DynamicFeeTx{
				ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
				Nonce:     nonce,
				GasTipCap: t.config.TipCap,
				GasFeeCap: t.config.FeeCap,
				Gas:       t.config.GasLimit,
				To:        toAddr,
				Value:     txAmount,
				Data:      txData,
			}
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

	err = txWallet.SendTransaction(ctx, tx, &wallet.SendTransactionOptions{
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

			if retry > 0 {
				logEntry = logEntry.WithField("retry", retry)
			}

			logEntry.Infof("submitted tx %v: %v", transactionIdx, tx.Hash().Hex())
		},
		RebroadcastInterval: 30 * time.Second,
		MaxRebroadcasts:     5,
	})

	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "nonce") {
			txWallet.ResyncState()
		}

		return err
	}

	return nil
}
