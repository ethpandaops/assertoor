package generateblobtransactions

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
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
	TaskName       = "generate_blob_transactions"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates blob transactions and sends them to the network",
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

func (t *Task) Config() any {
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

	perBlockCount := 0
	totalCount := 0

	t.ctx.ReportProgress(0, "Generating blob transactions...")

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

			// Note: onComplete callback is still called by spamoor even on error,
			// so we don't drain pendingChan here
		} else {
			perBlockCount++
			totalCount++

			if t.config.LimitTotal > 0 {
				progress := float64(totalCount) / float64(t.config.LimitTotal) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d blob transactions", totalCount, t.config.LimitTotal))
			} else {
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d blob transactions", totalCount))
			}
		}

		if t.config.LimitTotal > 0 && totalCount >= t.config.LimitTotal {
			t.ctx.ReportProgress(100, fmt.Sprintf("Completed: generated %d blob transactions", totalCount))

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

func (t *Task) generateTransaction(ctx context.Context, transactionIdx uint64, completeFn spamoor.TxCompleteFn) error {
	txWallet := t.wallet
	if t.wallet == nil {
		txWallet = t.walletPool.GetWallet(spamoor.SelectWalletRoundRobin, 0)
	}

	blobCount := t.config.BlobSidecars

	// Parse blobData formats:
	//   Old format (per-sidecar): "refs;refs;refs" — semicolons separate sidecar groups
	//   New format (all sidecars): "ref,ref,ref" — commas separate refs, applied to all sidecars
	// "identifier" and "label" are replaced with a unique blob label.
	blobData := t.config.BlobData
	if blobData == "" {
		blobData = "identifier,random"
	}

	var blobDataGroups []string

	if strings.Contains(blobData, ";") {
		blobDataGroups = strings.Split(blobData, ";")
		blobCount = uint64(len(blobDataGroups))
	} else {
		for range blobCount {
			blobDataGroups = append(blobDataGroups, blobData)
		}
	}

	blobRefs := make([][]string, blobCount)

	for i := uint64(0); i < blobCount; i++ {
		blobLabel := fmt.Sprintf("0x1611BB0000%08dFF%02dFF%04dFEED", 0, i, 0)
		blobRefs[i] = []string{}

		for blob := range strings.SplitSeq(blobDataGroups[i], ",") {
			if blob == "identifier" || blob == "label" {
				blob = blobLabel
			}

			blobRefs[i] = append(blobRefs[i], blob)
		}
	}

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
		n, err := rand.Int(rand.Reader, txAmount)
		if err == nil {
			txAmount = n
		}
	}

	txData := []byte{}
	if t.transactionData != nil {
		txData = t.transactionData
	}

	blobTx, err := txbuilder.BuildBlobTx(&txbuilder.TxMetadata{
		GasFeeCap:  uint256.MustFromBig(t.config.FeeCap),
		GasTipCap:  uint256.MustFromBig(t.config.TipCap),
		BlobFeeCap: uint256.MustFromBig(t.config.BlobFeeCap),
		Gas:        t.config.GasLimit,
		To:         &toAddr,
		Value:      uint256.MustFromBig(txAmount),
		Data:       txData,
	}, blobRefs)
	if err != nil {
		return fmt.Errorf("cannot build blob tx data: %w", err)
	}

	tx, err := txWallet.BuildBlobTx(blobTx)
	if err != nil {
		return fmt.Errorf("cannot build blob transaction: %w", err)
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

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()
	client := walletMgr.GetClient(clients[transactionIdx%uint64(len(clients))])

	return walletMgr.GetTxPool().SendTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      client,
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

			logEntry.Infof("submitted blob transaction %v (nonce: %v, attempt: %v)", transactionIdx, tx.Nonce(), retry)
		},
	})
}
