package generateeoatransactions

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
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

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
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
		t.wallet, err = t.ctx.Scheduler.GetCoordinator().WalletManager().GetWalletByPrivkey(privKey)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet: %w", err)
		}
	} else {
		t.walletPool, err = t.ctx.Scheduler.GetCoordinator().WalletManager().GetWalletPoolByPrivkey(privKey, config.ChildWallets, config.WalletSeed)
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
	if config.TransactionData != "" {
		t.transactionData = common.FromHex(config.TransactionData)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.walletPool != nil {
		err := t.ensureChildWalletFunding(ctx)
		if err != nil {
			t.logger.Infof("failed ensuring child wallet funding: %v", err)
			return err
		}

		for idx, wallet := range t.walletPool.GetChildWallets() {
			t.logger.Infof("wallet #%v: %v [nonce: %v]", idx, wallet.GetAddress().Hex(), wallet.GetNonce())
		}

		go t.runChildWalletFundingRoutine(ctx)
	}

	var subscription *execution.Subscription[*execution.Block]
	if t.config.LimitPerBlock > 0 {
		subscription = t.ctx.Scheduler.GetCoordinator().ClientPool().GetExecutionPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	var pendingChan chan bool

	if t.config.LimitPending > 0 {
		pendingChan = make(chan bool, t.config.LimitPending)
	}

	perBlockCount := 0
	totalCount := 0

	for {
		if pendingChan != nil {
			select {
			case <-ctx.Done():
				return nil
			case pendingChan <- true:
			}
		}

		txIndex := t.txIndex
		t.txIndex++

		err := t.generateTransaction(ctx, txIndex, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt) {
			if pendingChan != nil {
				<-pendingChan
			}

			if receipt != nil {
				t.logger.Infof("transaction %v confirmed (nonce: %v, status: %v)", tx.Hash().Hex(), tx.Nonce(), receipt.Status)
			} else {
				t.logger.Infof("transaction %v replaced (nonce: %v)", tx.Hash().Hex(), tx.Nonce())
			}
		})
		if err != nil {
			t.logger.Errorf("error generating transaction: %v", err.Error())
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

func (t *Task) generateTransaction(ctx context.Context, transactionIdx uint64, confirmedFn func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt)) error {
	txWallet := t.wallet
	if t.wallet == nil {
		txWallet = t.walletPool.GetNextChildWallet()
	}

	tx, err := txWallet.BuildTransaction(ctx, func(ctx context.Context, nonce uint64, signer bind.SignerFn) (*ethtypes.Transaction, error) {
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

		txAmount := new(big.Int).Set(t.config.TransactionAmount)
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

		if t.config.LegacyTransactions {
			txObj = &ethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: t.config.TransactionFeeCap,
				Gas:      t.config.TransactionGasLimit,
				To:       toAddr,
				Value:    txAmount,
				Data:     txData,
			}
		} else {
			txObj = &ethtypes.DynamicFeeTx{
				ChainID:   t.ctx.Scheduler.GetCoordinator().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
				Nonce:     nonce,
				GasTipCap: t.config.TransactionTipCap,
				GasFeeCap: t.config.TransactionFeeCap,
				Gas:       t.config.TransactionGasLimit,
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

	clientPool := t.ctx.Scheduler.GetCoordinator().ClientPool()

	if t.config.ClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints()
	} else {
		poolClients := clientPool.GetClientsByNamePatterns([]string{t.config.ClientPattern})
		if len(poolClients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	client := clients[transactionIdx%uint64(len(clients))]

	t.logger.WithFields(logrus.Fields{
		"client": client.GetName(),
	}).Infof("sending tx %v: %v", transactionIdx, tx.Hash().Hex())

	err = client.GetRPCClient().SendTransaction(ctx, tx)
	if err != nil {
		return err
	}

	go func() {
		receipt, err := txWallet.AwaitTransaction(ctx, tx)
		if err != nil {
			t.logger.Warnf("failed waiting for tx receipt: %v", err)
		}

		confirmedFn(tx, receipt)
	}()

	return nil
}
