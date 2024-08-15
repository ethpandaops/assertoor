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
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"

	depositcontract "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_deposits/deposit_contract"
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

	t.valkeySeed, err = t.mnemonicToSeed(config.Mnemonic)
	if err != nil {
		return err
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
		t.nextIndex = uint64(t.config.StartIndex)
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount)
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

		pubkey, tx, err := t.generateDeposit(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt) {
			if pendingChan != nil {
				<-pendingChan
			}

			pendingWg.Done()

			depositReceiptsMtx.Lock()
			depositReceipts[tx.Hash().Hex()] = receipt
			depositReceiptsMtx.Unlock()

			if receipt != nil {
				t.logger.Infof("deposit %v confirmed in block %v (nonce: %v, status: %v)", tx.Hash().Hex(), receipt.BlockNumber, tx.Nonce(), receipt.Status)
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

func (t *Task) generateDeposit(ctx context.Context, accountIdx uint64, onConfirm func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt)) (*common.BLSPubkey, *ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)
	withdrAccPath := fmt.Sprintf("m/12381/3600/%d/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.valkeySeed, validatorKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()

	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()
	for _, val := range validatorSet {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if validator != nil {
		return nil, nil, fmt.Errorf("validator already exists on chain")
	}

	var pub common.BLSPubkey

	var withdrCreds []byte

	copy(pub[:], validatorPubkey)

	if t.config.WithdrawalCredentials == "" {
		withdrPrivkey, err2 := util.PrivateKeyFromSeedAndPath(t.valkeySeed, withdrAccPath)
		if err2 != nil {
			return nil, nil, fmt.Errorf("failed generating key %v: %w", withdrAccPath, err2)
		}

		var withdrPub common.BLSPubkey

		copy(withdrPub[:], withdrPrivkey.PublicKey().Marshal())

		withdrCreds = withdrPub[:]
		withdrCreds[0] = common.BLS_WITHDRAWAL_PREFIX
	} else {
		withdrCreds = ethcommon.FromHex(t.config.WithdrawalCredentials)
	}

	data := common.DepositData{
		Pubkey:                pub,
		WithdrawalCredentials: tree.Root(withdrCreds),
		Amount:                common.Gwei(t.config.DepositAmount * 1000000000),
		Signature:             common.BLSSignature{},
	}
	msgRoot := data.ToMessage().HashTreeRoot(tree.GetHashFn())

	var secKey hbls.SecretKey

	err = secKey.Deserialize(validatorPrivkey.Marshal())
	if err != nil {
		return nil, nil, fmt.Errorf("cannot convert validator priv key")
	}

	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(common.DOMAIN_DEPOSIT, common.Version(genesis.GenesisForkVersion), common.Root{})
	msg := common.ComputeSigningRoot(msgRoot, dom)
	sig := secKey.SignHash(msg[:])
	copy(data.Signature[:], sig.Serialize())

	dataRoot := data.HashTreeRoot(tree.GetHashFn())

	// generate deposit transaction

	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints()
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

	wallet, err := t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.walletPrivKey)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	err = wallet.AwaitReady(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load wallet state: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", wallet.GetAddress().Hex(), wallet.GetNonce(), wallet.GetReadableBalance(18, 0, 4, false, false))

	tx, err := wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, signer bind.SignerFn) (*ethtypes.Transaction, error) {
		amount := big.NewInt(int64(data.Amount))

		amount.Mul(amount, big.NewInt(1000000000))

		return depositContract.Deposit(&bind.TransactOpts{
			From:      wallet.GetAddress(),
			Nonce:     big.NewInt(int64(nonce)),
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

	for i := 0; i < len(clients); i++ {
		client := clients[i%len(clients)]

		t.logger.WithFields(logrus.Fields{
			"client": client.GetName(),
		}).Infof("sending deposit transaction (account idx: %v, nonce: %v)", accountIdx, tx.Nonce())

		err = client.GetRPCClient().SendTransaction(ctx, tx)
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed sending deposit transaction: %w", err)
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

	return &pub, tx, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
