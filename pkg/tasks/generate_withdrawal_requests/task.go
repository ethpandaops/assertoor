package generatewithdrawalrequests

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
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
	TaskName       = "generate_withdrawal_requests"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates withdrawal requests and sends them to the network",
		Category:    "validator",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "transactionHashes",
				Type:        "array",
				Description: "Array of withdrawal transaction hashes.",
			},
			{
				Name:        "transactionReceipts",
				Type:        "array",
				Description: "Array of withdrawal transaction receipts.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx                    *types.TaskContext
	options                *types.TaskOptions
	config                 Config
	logger                 logrus.FieldLogger
	sourceSeed             []byte
	nextIndex              uint64
	lastIndex              uint64
	walletPrivKey          *ecdsa.PrivateKey
	withdrawalContractAddr ethcommon.Address
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

	t.withdrawalContractAddr = ethcommon.HexToAddress(config.WithdrawalContract)

	t.config = config

	return nil
}

//nolint:gocyclo // no need to reduce complexity
func (t *Task) Execute(ctx context.Context) error {
	if t.config.SourceStartIndex > 0 {
		t.nextIndex = uint64(t.config.SourceStartIndex) //nolint:gosec // no overflow possible
	}

	if t.config.SourceIndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.SourceIndexCount) //nolint:gosec // no overflow possible
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

	t.ctx.ReportProgress(0, "Generating withdrawal requests...")

	withdrawalTransactions := []string{}
	receiptsMapMutex := sync.Mutex{}
	withdrawalReceipts := map[string]*ethtypes.Receipt{}

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

		tx, err := t.generateWithdrawal(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
			if pendingChan != nil {
				<-pendingChan
			}

			receiptsMapMutex.Lock()

			withdrawalReceipts[tx.Hash().Hex()] = receipt

			receiptsMapMutex.Unlock()
			pendingWg.Done()

			switch {
			case receipt != nil:
				t.logger.Infof("withdrawal %v confirmed (nonce: %v, status: %v)", tx.Hash().Hex(), tx.Nonce(), receipt.Status)
			case err != nil:
				t.logger.Errorf("error awaiting withdrawal transaction receipt: %v", err.Error())
			default:
				t.logger.Warnf("no receipt for withdrawal transaction: %v", tx.Hash().Hex())
			}
		})
		if err != nil {
			t.logger.Errorf("error generating withdrawal: %v", err.Error())

			// Note: onComplete callback is still called by spamoor even on error,
			// so we don't drain pendingChan or call pendingWg.Done() here
		} else {
			t.ctx.SetResult(types.TaskResultSuccess)

			perSlotCount++
			totalCount++

			withdrawalTransactions = append(withdrawalTransactions, tx.Hash().Hex())

			// Report progress based on total limit or index count
			switch {
			case t.config.LimitTotal > 0:
				progress := float64(totalCount) / float64(t.config.LimitTotal) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d withdrawal requests", totalCount, t.config.LimitTotal))
			case t.lastIndex > 0:
				indexTotal := t.lastIndex - uint64(t.config.SourceStartIndex) //nolint:gosec // no overflow possible
				progress := float64(totalCount) / float64(indexTotal) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d withdrawal requests", totalCount, indexTotal))
			default:
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d withdrawal requests", totalCount))
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

	t.ctx.ReportProgress(100, fmt.Sprintf("Completed: generated %d withdrawal requests", totalCount))

	t.ctx.Outputs.SetVar("transactionHashes", withdrawalTransactions)

	receiptList := []interface{}{}

	receiptsMapMutex.Lock()
	defer receiptsMapMutex.Unlock()

	for _, txhash := range withdrawalTransactions {
		var receiptMap map[string]interface{}

		receipt := withdrawalReceipts[txhash]
		if receipt == nil {
			receiptMap = nil
		} else {
			receiptJSON, err := json.Marshal(receipt)
			if err == nil {
				receiptMap = map[string]interface{}{}

				err = json.Unmarshal(receiptJSON, &receiptMap)
				if err != nil {
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
		for _, txhash := range withdrawalTransactions {
			if withdrawalReceipts[txhash] == nil {
				t.logger.Errorf("no receipt for withdrawal transaction: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}

			if withdrawalReceipts[txhash].Status == 0 {
				t.logger.Errorf("withdrawal transaction failed: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}
		}
	}

	return nil
}

func (t *Task) generateWithdrawal(ctx context.Context, accountIdx uint64, onComplete spamoor.TxCompleteFn) (*ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	var sourcePubkey []byte

	if t.config.SourcePubkey != "" {
		sourcePubkey = ethcommon.FromHex(t.config.SourcePubkey)
	} else {
		var sourceValidator *v1.Validator

		validatorSet := clientPool.GetConsensusPool().GetValidatorSet()
		sourceSelector := ""

		if t.sourceSeed != nil {
			// select by key index
			validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

			validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.sourceSeed, validatorKeyPath)
			if err != nil {
				return nil, fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
			}

			sourceValidatorPubkey := validatorPrivkey.PublicKey().Marshal()
			sourceSelector = fmt.Sprintf("(pubkey: 0x%x)", sourceValidatorPubkey)

			for _, val := range validatorSet {
				if bytes.Equal(val.Validator.PublicKey[:], sourceValidatorPubkey) {
					sourceValidator = val
					break
				}
			}
		} else if t.config.SourceStartValidatorIndex != nil {
			// select by validator index
			validatorIndex := *t.config.SourceStartValidatorIndex + accountIdx
			sourceSelector = fmt.Sprintf("(index: %v)", validatorIndex)
			sourceValidator = validatorSet[phase0.ValidatorIndex(validatorIndex)]
		}

		if sourceValidator == nil {
			return nil, fmt.Errorf("source validator %s not found in validator set", sourceSelector)
		}

		sourcePubkey = sourceValidator.Validator.PublicKey[:]
	}

	// generate withdrawal transaction

	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints(true)
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

	if len(clients) == 0 {
		return nil, fmt.Errorf("no ready clients available")
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	txWallet, err := walletMgr.GetWalletByPrivkey(ctx, t.walletPrivKey)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	amount := big.NewInt(0).SetUint64(t.config.WithdrawAmount)
	amountBytes := amount.FillBytes(make([]byte, 8))

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", txWallet.GetAddress().Hex(), txWallet.GetNonce(), txWallet.GetReadableBalance(18, 0, 4, false, false))

	txData := make([]byte, 56) // 48 bytes pubkey + 8 bytes amount
	copy(txData[0:48], sourcePubkey)
	copy(txData[48:], amountBytes)

	dynFeeTx, err := txbuilder.DynFeeTx(&txbuilder.TxMetadata{
		GasTipCap: uint256.MustFromBig(t.config.TxTipCap),
		GasFeeCap: uint256.MustFromBig(t.config.TxFeeCap),
		Gas:       t.config.TxGasLimit,
		To:        &t.withdrawalContractAddr,
		Value:     uint256.MustFromBig(t.config.TxAmount),
		Data:      txData,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build withdrawal tx data: %w", err)
	}

	tx, err := txWallet.BuildDynamicFeeTx(dynFeeTx)
	if err != nil {
		return nil, fmt.Errorf("cannot build withdrawal transaction: %w", err)
	}

	client := walletMgr.GetClient(clients[0])

	err = walletMgr.GetTxPool().SendTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      client,
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

			logEntry.Infof("submitted withdrawal transaction (source pubkey: 0x%x, amount: %v, nonce: %v)", sourcePubkey, amount.String(), tx.Nonce())
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed sending withdrawal transaction: %w", err)
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
