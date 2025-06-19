package generatedeposits

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
	"github.com/erigontech/assertoor/pkg/coordinator/clients/consensus"
	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/util/hashing"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	e2types "github.com/wealdtech/go-eth2-types/v2"
	util "github.com/wealdtech/go-eth2-util"

	depositcontract "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_deposits/deposit_contract"
)

var (
	TaskName       = "generate_deposits"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates deposits and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx                 *types.TaskContext
	options             *types.TaskOptions
	config              Config
	logger              logrus.FieldLogger
	valkeySeed          []byte
	nextIndex           uint64
	lastIndex           uint64
	walletPrivKey       *ecdsa.PrivateKey
	depositContractAddr ethcommon.Address
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

	if config.Mnemonic != "" {
		t.valkeySeed, err = t.mnemonicToSeed(config.Mnemonic)
		if err != nil {
			return err
		}

		t.logger.Infof("validator key seed: 0x%x", t.valkeySeed)
	}

	t.walletPrivKey, err = crypto.HexToECDSA(config.WalletPrivkey)
	if err != nil {
		return err
	}

	t.config = config
	t.depositContractAddr = ethcommon.HexToAddress(config.DepositContract)

	return nil
}

//nolint:gocyclo // ignore
func (t *Task) Execute(ctx context.Context) error {
	if t.config.StartIndex > 0 {
		t.nextIndex = uint64(t.config.StartIndex) //nolint:gosec // no overflow possible
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount) //nolint:gosec // no overflow possible
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

	depositTransactions := []string{}
	validatorPubkeys := []string{}
	depositReceipts := map[string]*ethtypes.Receipt{}
	depositReceiptsMtx := sync.Mutex{}

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

		pubkey, tx, err := t.generateDeposit(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
			if pendingChan != nil {
				<-pendingChan
			}

			depositReceiptsMtx.Lock()
			depositReceipts[tx.Hash().Hex()] = receipt
			depositReceiptsMtx.Unlock()

			switch {
			case receipt != nil:
				t.logger.Infof("deposit %v confirmed in block %v (nonce: %v, status: %v)", tx.Hash().Hex(), receipt.BlockNumber, tx.Nonce(), receipt.Status)
			case err != nil:
				t.logger.Errorf("error awaiting deposit transaction receipt: %v", err.Error())
			default:
				t.logger.Warnf("no receipt for deposit transaction: %v", tx.Hash().Hex())
			}

			pendingWg.Done()
		})
		if err != nil {
			t.logger.Errorf("error generating deposit: %v", err.Error())

			if pendingChan != nil {
				<-pendingChan
			}

			pendingWg.Done()
		} else {
			t.ctx.SetResult(types.TaskResultSuccess)

			perSlotCount++
			totalCount++

			validatorPubkeys = append(validatorPubkeys, pubkey.String())
			depositTransactions = append(depositTransactions, tx.Hash().Hex())
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

	if t.config.ValidatorPubkeysResultVar != "" {
		t.ctx.Vars.SetVar(t.config.ValidatorPubkeysResultVar, validatorPubkeys)
	}

	t.ctx.Outputs.SetVar("validatorPubkeys", validatorPubkeys)

	if t.config.DepositTransactionsResultVar != "" {
		t.ctx.Vars.SetVar(t.config.DepositTransactionsResultVar, depositTransactions)
	}

	t.ctx.Outputs.SetVar("depositTransactions", depositTransactions)

	receiptList := []interface{}{}

	for _, txhash := range depositTransactions {
		var receiptMap map[string]interface{}

		receipt := depositReceipts[txhash]
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

	if t.config.DepositReceiptsResultVar != "" {
		t.ctx.Vars.SetVar(t.config.DepositReceiptsResultVar, receiptList)
	}

	t.ctx.Outputs.SetVar("depositReceipts", receiptList)

	if t.config.FailOnReject {
		for _, txhash := range depositTransactions {
			if depositReceipts[txhash] == nil {
				t.logger.Errorf("no receipt for deposit transaction: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}

			if depositReceipts[txhash].Status == 0 {
				t.logger.Errorf("deposit transaction failed: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}
		}
	}

	return nil
}

func (t *Task) generateDeposit(ctx context.Context, accountIdx uint64, onConfirm wallet.TxConfirmFn) (*common.BLSPubkey, *ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()

	var validatorPubkey []byte

	var validatorPrivkey *e2types.BLSPrivateKey

	if t.valkeySeed != nil {
		validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

		validatorPriv, err := util.PrivateKeyFromSeedAndPath(t.valkeySeed, validatorKeyPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
		}

		validatorPrivkey = validatorPriv

		validatorPubkey = validatorPrivkey.PublicKey().Marshal()
		t.logger.Debugf("generated validator pubkey %v: 0x%x", validatorKeyPath, validatorPubkey)
	} else {
		validatorPubkey = ethcommon.FromHex(t.config.PublicKey)
	}

	var validator *v1.Validator

	for _, val := range validatorSet {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if t.valkeySeed != nil && validator != nil {
		t.logger.Warnf("validator already exists on chain (index: %v)", validator.Index)
	} else if t.valkeySeed == nil && validator == nil {
		t.logger.Warnf("validator not found on chain for topup deposit")
	}

	var pub common.BLSPubkey

	var withdrCreds []byte

	copy(pub[:], validatorPubkey)

	switch {
	case t.config.WithdrawalCredentials != "":
		withdrCreds = ethcommon.FromHex(t.config.WithdrawalCredentials)
	case t.config.TopUpDeposit:
		withdrCreds = ethcommon.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000")
	default:
		withdrAccPath := fmt.Sprintf("m/12381/3600/%d/0", accountIdx)

		withdrPrivkey, err2 := util.PrivateKeyFromSeedAndPath(t.valkeySeed, withdrAccPath)
		if err2 != nil {
			return nil, nil, fmt.Errorf("failed generating key %v: %w", withdrAccPath, err2)
		}

		withdrPubKey := withdrPrivkey.PublicKey().Marshal()
		t.logger.Debugf("generated withdrawal pubkey %v: 0x%x", withdrAccPath, withdrPubKey)

		withdrKeyHash := hashing.Hash(withdrPubKey)
		withdrCreds = withdrKeyHash[:]
		withdrCreds[0] = common.BLS_WITHDRAWAL_PREFIX
	}

	data := common.DepositData{
		Pubkey:                pub,
		WithdrawalCredentials: tree.Root(withdrCreds),
		Amount:                common.Gwei(t.config.DepositAmount * 1000000000),
		Signature:             common.BLSSignature{},
	}

	if !t.config.TopUpDeposit {
		msgRoot := data.ToMessage().HashTreeRoot(tree.GetHashFn())

		var secKey hbls.SecretKey

		err := secKey.Deserialize(validatorPrivkey.Marshal())
		if err != nil {
			return nil, nil, fmt.Errorf("cannot convert validator priv key")
		}

		genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
		dom := common.ComputeDomain(common.DOMAIN_DEPOSIT, common.Version(genesis.GenesisForkVersion), common.Root{})
		msg := common.ComputeSigningRoot(msgRoot, dom)
		sig := secKey.SignHash(msg[:])
		copy(data.Signature[:], sig.Serialize())
	}

	dataRoot := data.HashTreeRoot(tree.GetHashFn())

	// generate deposit transaction

	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints(true)
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			return nil, nil, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	if len(clients) == 0 {
		return nil, nil, fmt.Errorf("no ready clients available")
	}

	depositContract, err := depositcontract.NewDepositContract(t.depositContractAddr, clients[0].GetRPCClient().GetEthClient())
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create bound instance of DepositContract: %w", err)
	}

	txWallet, err := t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.walletPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	err = txWallet.AwaitReady(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load wallet state: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", txWallet.GetAddress().Hex(), txWallet.GetNonce(), txWallet.GetReadableBalance(18, 0, 4, false, false))

	tx, err := txWallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, signer bind.SignerFn) (*ethtypes.Transaction, error) {
		amount := big.NewInt(0).SetUint64(uint64(data.Amount))

		amount.Mul(amount, big.NewInt(1000000000))

		return depositContract.Deposit(&bind.TransactOpts{
			From:      txWallet.GetAddress(),
			Nonce:     big.NewInt(0).SetUint64(nonce),
			Value:     amount,
			GasLimit:  200000,
			GasFeeCap: big.NewInt(t.config.DepositTxFeeCap),
			GasTipCap: big.NewInt(t.config.DepositTxTipCap),
			Signer:    signer,
			NoSend:    true,
		}, data.Pubkey[:], data.WithdrawalCredentials[:], data.Signature[:], dataRoot)
	})
	if err != nil {
		return nil, nil, fmt.Errorf("cannot build deposit transaction: %w", err)
	}

	err = txWallet.SendTransaction(ctx, tx, &wallet.SendTransactionOptions{
		Clients:   clients,
		OnConfirm: onConfirm,
		LogFn: func(client *execution.Client, retry uint64, rebroadcast uint64, err error) {
			if err != nil {
				return
			}

			logEntry := t.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
			})

			if rebroadcast > 0 {
				logEntry = logEntry.WithField("rebroadcast", rebroadcast)
			}

			logEntry.Infof("submitted deposit transaction (account idx: %v, nonce: %v, attempt: %v)", accountIdx, tx.Nonce(), retry)
		},
		RebroadcastInterval: 30 * time.Second,
		MaxRebroadcasts:     5,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed sending deposit transaction: %w", err)
	}

	return &pub, tx, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
