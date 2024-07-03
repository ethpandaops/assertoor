package generateconsolidations

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_consolidations"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates consolidations and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

var DomainConsolidation = common.BLSDomainType{0x0B, 0x00, 0x00, 0x00}

type Task struct {
	ctx                       *types.TaskContext
	options                   *types.TaskOptions
	config                    Config
	logger                    logrus.FieldLogger
	sourceSeed                []byte
	nextIndex                 uint64
	lastIndex                 uint64
	walletPrivKey             *ecdsa.PrivateKey
	consolidationContractAddr ethcommon.Address
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

	if config.SourceMnemonic != "" {
		t.sourceSeed, err = t.mnemonicToSeed(config.SourceMnemonic)
		if err != nil {
			return err
		}
	}

	t.walletPrivKey, err = crypto.HexToECDSA(config.WalletPrivkey)
	if err != nil {
		return err
	}

	t.consolidationContractAddr = ethcommon.HexToAddress(config.ConsolidationContract)

	t.config = config

	return nil
}

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

	consolidationTransactions := []string{}
	consolidationReceipts := map[string]*ethtypes.Receipt{}

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		tx, err := t.generateConsolidation(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt) {
			if pendingChan != nil {
				<-pendingChan
			}

			pendingWg.Done()

			consolidationReceipts[tx.Hash().Hex()] = receipt

			if receipt != nil {
				t.logger.Infof("deposit %v confirmed (nonce: %v, status: %v)", tx.Hash().Hex(), tx.Nonce(), receipt.Status)
			}
		})
		if err != nil {
			t.logger.Errorf("error generating deposit: %v", err.Error())
		} else {
			if pendingChan != nil {
				select {
				case <-ctx.Done():
					return nil
				case pendingChan <- true:
				}
			}

			pendingWg.Add(1)

			t.ctx.SetResult(types.TaskResultSuccess)

			perSlotCount++
			totalCount++

			consolidationTransactions = append(consolidationTransactions, tx.Hash().Hex())
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

	if t.config.ConsolidationTransactionsResultVar != "" {
		t.ctx.Vars.SetVar(t.config.ConsolidationTransactionsResultVar, consolidationTransactions)
	}

	if t.config.ConsolidationReceiptsResultVar != "" {
		receiptList := []interface{}{}

		for _, txhash := range consolidationTransactions {
			var receiptMap map[string]interface{}

			receipt := consolidationReceipts[txhash]
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

		t.ctx.Vars.SetVar(t.config.ConsolidationReceiptsResultVar, receiptList)
	}

	if t.config.FailOnReject {
		for _, txhash := range consolidationTransactions {
			if consolidationReceipts[txhash] == nil {
				t.logger.Errorf("no receipt for consolidation transaction: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}

			if consolidationReceipts[txhash].Status == 0 {
				t.logger.Errorf("consolidation transaction failed: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}
		}
	}

	return nil
}

func (t *Task) generateConsolidation(ctx context.Context, accountIdx uint64, onConfirm func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt)) (*ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	var sourceValidator, targetValidator *v1.Validator
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

	targetValidator = validatorSet[phase0.ValidatorIndex(*t.config.TargetValidatorIndex)]
	if targetValidator == nil {
		return nil, fmt.Errorf("target validator (index: %v) not found", *t.config.TargetValidatorIndex)
	}

	// generate consolidation transaction

	var client *execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetExecutionPool().AwaitReadyEndpoint(ctx, execution.AnyClient)
		if client == nil {
			return nil, ctx.Err()
		}
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return nil, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		client = clients[0].ExecutionClient
	}

	wallet, err := t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.walletPrivKey)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	err = wallet.AwaitReady(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load wallet state: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", wallet.GetAddress().Hex(), wallet.GetNonce(), wallet.GetReadableBalance(18, 0, 4, false, false))

	tx, err := wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		txData := make([]byte, 96)
		copy(txData[0:48], sourceValidator.Validator.PublicKey[:])
		copy(txData[48:], targetValidator.Validator.PublicKey[:])

		txObj := &ethtypes.DynamicFeeTx{
			ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: t.config.TxTipCap,
			GasFeeCap: t.config.TxFeeCap,
			Gas:       t.config.TxGasLimit,
			To:        &t.consolidationContractAddr,
			Value:     t.config.TxAmount,
			Data:      txData,
		}

		return ethtypes.NewTx(txObj), nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build consolidation transaction: %w", err)
	}

	t.logger.Infof("sending consolidation transaction (source index: %v, target index: %v, nonce: %v)", sourceValidator.Index, targetValidator.Index, tx.Nonce())

	err = client.GetRPCClient().SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed sending consolidation transaction: %w", err)
	}

	go func() {
		var receipt *ethtypes.Receipt

		if onConfirm != nil {
			defer func() {
				onConfirm(tx, receipt)
			}()
		}

		receipt, err := wallet.AwaitTransaction(ctx, tx)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			t.logger.Errorf("failed awaiting transaction receipt: %w", err)
			return
		}
	}()

	return tx, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
