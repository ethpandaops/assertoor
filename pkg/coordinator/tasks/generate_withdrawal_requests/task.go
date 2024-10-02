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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_withdrawal_requests"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates withdrawal requests and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx                    *types.TaskContext
	options                *types.TaskOptions
	config                 Config
	logger                 logrus.FieldLogger
	sourceSeed             []byte
	nextIndex              uint64
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
			return err
		}
	}

	t.walletPrivKey, err = crypto.HexToECDSA(config.WalletPrivkey)
	if err != nil {
		return err
	}

	t.withdrawalContractAddr = ethcommon.HexToAddress(config.WithdrawalContract)

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.config.SourceStartIndex > 0 {
		t.nextIndex = uint64(t.config.SourceStartIndex)
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

	withdrawalTransactions := []string{}
	withdrawalReceipts := map[string]*ethtypes.Receipt{}

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		tx, err := t.generateWithdrawal(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt) {
			if pendingChan != nil {
				<-pendingChan
			}

			withdrawalReceipts[tx.Hash().Hex()] = receipt

			pendingWg.Done()

			if receipt != nil {
				t.logger.Infof("withdrawal %v confirmed (nonce: %v, status: %v)", tx.Hash().Hex(), tx.Nonce(), receipt.Status)
			}
		})
		if err != nil {
			t.logger.Errorf("error generating withdrawal: %v", err.Error())
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

			withdrawalTransactions = append(withdrawalTransactions, tx.Hash().Hex())
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

	t.ctx.Outputs.SetVar("transactionHashes", withdrawalTransactions)

	receiptList := []interface{}{}

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

func (t *Task) generateWithdrawal(ctx context.Context, accountIdx uint64, onConfirm func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt)) (*ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	var sourceValidator *v1.Validator

	if t.config.SourceIndexCount > 0 {
		accountIdx %= uint64(t.config.SourceIndexCount)
	}

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

	// generate withdrawal transaction

	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints()
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

	wallet, err := t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.walletPrivKey)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	err = wallet.AwaitReady(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load wallet state: %w", err)
	}

	amount := big.NewInt(0).SetUint64(t.config.WithdrawAmount)
	amountBytes := amount.FillBytes(make([]byte, 16))

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", wallet.GetAddress().Hex(), wallet.GetNonce(), wallet.GetReadableBalance(18, 0, 4, false, false))

	tx, err := wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
		txData := make([]byte, 64) // 48 bytes pubkey + 16 bytes amount
		copy(txData[0:48], sourceValidator.Validator.PublicKey[:])
		copy(txData[48:], amountBytes)

		txObj := &ethtypes.DynamicFeeTx{
			ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
			Nonce:     nonce,
			GasTipCap: t.config.TxTipCap,
			GasFeeCap: t.config.TxFeeCap,
			Gas:       t.config.TxGasLimit,
			To:        &t.withdrawalContractAddr,
			Value:     t.config.TxAmount,
			Data:      txData,
		}

		return ethtypes.NewTx(txObj), nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build withdrawal transaction: %w", err)
	}

	for i := 0; i < len(clients); i++ {
		client := clients[i%len(clients)]

		t.logger.WithFields(logrus.Fields{
			"client": client.GetName(),
		}).Infof("sending withdrawal transaction (source index: %v, amount: %v, nonce: %v)", sourceValidator.Index, amount.String(), tx.Nonce())

		err = client.GetRPCClient().SendTransaction(ctx, tx)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed sending withdrawal transaction: %w", err)
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
