package generatebuilderexits

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/ethpandaops/spamoor/txbuilder"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_builder_exits"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates builder exit requests (EIP-8282) and sends them to the network",
		Category:    "validator",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "transactionHashes",
				Type:        "array",
				Description: "Array of builder exit transaction hashes.",
			},
			{
				Name:        "transactionReceipts",
				Type:        "array",
				Description: "Array of builder exit transaction receipts.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx              *types.TaskContext
	options          *types.TaskOptions
	config           Config
	logger           logrus.FieldLogger
	sourceSeed       []byte
	nextIndex        uint64
	lastIndex        uint64
	walletPrivKey    *ecdsa.PrivateKey
	exitContractAddr ethcommon.Address
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

	if config.SourceMnemonic != "" {
		t.sourceSeed, err = t.mnemonicToSeed(config.SourceMnemonic)
		if err != nil {
			return fmt.Errorf("failed parsing sourceMnemonic: %v", err)
		}
	}

	t.walletPrivKey, err = crypto.HexToECDSA(config.WalletPrivkey)
	if err != nil {
		return fmt.Errorf("failed parsing walletPrivKey: %v", err)
	}

	t.exitContractAddr = ethcommon.HexToAddress(config.BuilderExitContract)

	t.config = config

	return nil
}

//nolint:gocyclo // no need to reduce complexity
func (t *Task) Execute(ctx context.Context) error {
	if t.config.SourceStartIndex > 0 {
		t.nextIndex = uint64(t.config.SourceStartIndex)
	}

	if t.config.SourceIndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.SourceIndexCount)
	}

	var subscription *consensus.Subscription[*consensus.Block]

	if t.config.LimitPerSlot > 0 {
		subscription = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	var pendingChan chan bool

	pendingWg := sync.WaitGroup{}

	if t.config.LimitPending > 0 {
		pendingChan = make(chan bool, t.config.LimitPending)
	}

	perSlotCount := 0
	totalCount := 0

	t.ctx.ReportProgress(0, "Generating builder exit requests...")

	exitTransactions := []string{}
	receiptsMapMutex := sync.Mutex{}
	exitReceipts := map[string]*ethtypes.Receipt{}

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		if pendingChan != nil {
			select {
			case <-ctx.Done():
				return nil
			case pendingChan <- true:
			}
		}

		pendingWg.Add(1)

		tx, err := t.generateBuilderExit(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
			if pendingChan != nil {
				<-pendingChan
			}

			receiptsMapMutex.Lock()

			exitReceipts[tx.Hash().Hex()] = receipt

			receiptsMapMutex.Unlock()
			pendingWg.Done()

			switch {
			case receipt != nil:
				t.logger.Infof("builder exit %v confirmed (nonce: %v, status: %v)", tx.Hash().Hex(), tx.Nonce(), receipt.Status)
			case err != nil:
				t.logger.Errorf("error awaiting builder exit transaction receipt: %v", err.Error())
			default:
				t.logger.Warnf("no receipt for builder exit transaction: %v", tx.Hash().Hex())
			}
		})
		if err != nil {
			t.logger.Errorf("error generating builder exit: %v", err.Error())
			// Note: onComplete callback is still called by spamoor even on error,
			// so we don't drain pendingChan or call pendingWg.Done() here
		} else {
			perSlotCount++
			totalCount++

			exitTransactions = append(exitTransactions, tx.Hash().Hex())

			switch {
			case t.config.LimitTotal > 0:
				progress := float64(totalCount) / float64(t.config.LimitTotal) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d builder exit requests", totalCount, t.config.LimitTotal))
			case t.lastIndex > 0:
				indexTotal := t.lastIndex - uint64(t.config.SourceStartIndex) //nolint:gosec // G115: config value is validated non-negative
				progress := float64(totalCount) / float64(indexTotal) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d builder exit requests", totalCount, indexTotal))
			default:
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d builder exit requests", totalCount))
			}
		}

		if t.lastIndex > 0 && t.nextIndex >= t.lastIndex {
			break
		}

		if t.config.LimitTotal > 0 && totalCount >= t.config.LimitTotal {
			break
		}

		if t.config.LimitPerSlot > 0 && perSlotCount >= t.config.LimitPerSlot {
			// await next block
			perSlotCount = 0

			select {
			case <-ctx.Done():
				return nil
			case <-subscription.Channel():
			}
		} else if ctx.Err() != nil {
			return nil
		}
	}

	if t.config.AwaitReceipt {
		pendingWg.Wait()
	}

	t.ctx.ReportProgress(100, fmt.Sprintf("Completed: generated %d builder exit requests", totalCount))

	t.ctx.Outputs.SetVar("transactionHashes", exitTransactions)

	receiptList := []interface{}{}

	receiptsMapMutex.Lock()
	defer receiptsMapMutex.Unlock()

	for _, txhash := range exitTransactions {
		var receiptMap map[string]interface{}

		receipt := exitReceipts[txhash]
		if receipt != nil {
			receiptJSON, err := json.Marshal(receipt)
			if err == nil {
				receiptMap = map[string]interface{}{}

				if err = json.Unmarshal(receiptJSON, &receiptMap); err != nil {
					t.logger.Errorf("could not unmarshal transaction receipt for result var: %v", err)

					receiptMap = nil
				}
			} else {
				t.logger.Errorf("could not marshal transaction receipt for result var: %v", err)
			}
		}

		receiptList = append(receiptList, receiptMap)
	}

	t.ctx.Outputs.SetVar("transactionReceipts", receiptList)

	if t.config.FailOnReject {
		for _, txhash := range exitTransactions {
			if exitReceipts[txhash] == nil {
				t.logger.Errorf("no receipt for builder exit transaction: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}

			if exitReceipts[txhash].Status == 0 {
				t.logger.Errorf("builder exit transaction failed: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}
		}
	}

	return nil
}

func (t *Task) generateBuilderExit(ctx context.Context, accountIdx uint64, onComplete spamoor.TxCompleteFn) (*ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	var builderPubkey []byte

	if t.config.SourcePubkey != "" {
		builderPubkey = ethcommon.FromHex(t.config.SourcePubkey)
	} else {
		// derive builder key from mnemonic
		builderKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

		builderPrivkey, err := util.PrivateKeyFromSeedAndPath(t.sourceSeed, builderKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed generating builder key %v: %w", builderKeyPath, err)
		}

		builderPubkey = builderPrivkey.PublicKey().Marshal()
	}

	if len(builderPubkey) != 48 {
		return nil, fmt.Errorf("invalid builder pubkey length: %d (want 48)", len(builderPubkey))
	}

	// select clients
	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().AwaitReadyEndpoints(ctx, true)
		if len(clients) == 0 {
			return nil, ctx.Err()
		}
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			return nil, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	spamoorClients := make([]*spamoor.Client, len(clients))
	for i, c := range clients {
		spamoorClients[i] = walletMgr.GetClient(c)
	}

	txWallet, err := walletMgr.GetWalletByPrivkey(ctx, t.walletPrivKey)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", txWallet.GetAddress().Hex(), txWallet.GetNonce(), txWallet.GetReadableBalance(18, 0, 4, false, false))

	// Build the raw 48-byte builder exit calldata (EIP-8282, no function selector):
	// the contract prepends msg.sender as the source address when emitting the request.
	txData := make([]byte, 48)
	copy(txData, builderPubkey)

	dynFeeTx, err := txbuilder.DynFeeTx(&txbuilder.TxMetadata{
		GasTipCap: uint256.MustFromBig(t.config.TxTipCap),
		GasFeeCap: uint256.MustFromBig(t.config.TxFeeCap),
		Gas:       t.config.TxGasLimit,
		To:        &t.exitContractAddr,
		Value:     uint256.MustFromBig(t.config.TxAmount),
		Data:      txData,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build builder exit tx data: %w", err)
	}

	tx, err := txWallet.BuildDynamicFeeTx(dynFeeTx)
	if err != nil {
		return nil, fmt.Errorf("cannot build builder exit transaction: %w", err)
	}

	err = walletMgr.GetTxPool().SendTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      spamoorClients[0],
		ClientList:  spamoorClients,
		Rebroadcast: true,
		OnComplete:  onComplete,
		LogFn: func(client *spamoor.Client, _ int, rebroadcast int, err error) {
			if err != nil {
				return
			}

			logEntry := t.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
			})

			if rebroadcast > 0 {
				logEntry = logEntry.WithField("rebroadcast", rebroadcast)
			}

			logEntry.Infof("submitted builder exit transaction (builder pubkey: 0x%x, nonce: %v)", builderPubkey, tx.Nonce())
		},
	})
	if err != nil {
		txWallet.MarkSkippedNonce(tx.Nonce())
		return nil, fmt.Errorf("failed sending builder exit transaction: %w", err)
	}

	return tx, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
