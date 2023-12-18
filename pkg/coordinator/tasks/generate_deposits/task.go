package generatedeposits

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"strings"
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
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/zrnt/eth2/util/hashing"
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

func (t *Task) Execute(ctx context.Context) error {
	if t.config.StartIndex > 0 {
		t.nextIndex = uint64(t.config.StartIndex)
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount)
	}

	var subscription *consensus.Subscription[*consensus.Block]
	if t.config.LimitPerSlot > 0 {
		subscription = t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	validators, err := t.loadChainState(ctx)
	if err != nil {
		return err
	}

	perSlotCount := 0
	totalCount := 0

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		err := t.generateDeposit(ctx, accountIdx, validators, nil)
		if err != nil {
			t.logger.Errorf("error generating deposit: %v", err.Error())
		} else {
			perSlotCount++
			totalCount++
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
		} else if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func (t *Task) loadChainState(ctx context.Context) (map[phase0.ValidatorIndex]*v1.Validator, error) {
	client := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)

	validators, err := client.GetRPCClient().GetStateValidators(ctx, "head")
	if err != nil {
		return nil, err
	}

	return validators, nil
}

func (t *Task) generateDeposit(ctx context.Context, accountIdx uint64, validators map[phase0.ValidatorIndex]*v1.Validator, onConfirm func()) error {
	clientPool := t.ctx.Scheduler.GetCoordinator().ClientPool()
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)
	withdrAccPath := fmt.Sprintf("m/12381/3600/%d/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.valkeySeed, validatorKeyPath)
	if err != nil {
		return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	withdrPrivkey, err := util.PrivateKeyFromSeedAndPath(t.valkeySeed, withdrAccPath)
	if err != nil {
		return fmt.Errorf("failed generating key %v: %w", withdrAccPath, err)
	}

	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()
	for _, val := range validators {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if validator != nil {
		return fmt.Errorf("validator already exists on chain")
	}

	var pub, withdrPub common.BLSPubkey

	copy(pub[:], validatorPubkey)
	copy(withdrPub[:], withdrPrivkey.PublicKey().Marshal())

	withdrCreds := hashing.Hash(withdrPub[:])
	withdrCreds[0] = common.BLS_WITHDRAWAL_PREFIX

	data := common.DepositData{
		Pubkey:                pub,
		WithdrawalCredentials: withdrCreds,
		Amount:                configs.Mainnet.MAX_EFFECTIVE_BALANCE,
		Signature:             common.BLSSignature{},
	}
	msgRoot := data.ToMessage().HashTreeRoot(tree.GetHashFn())

	var secKey hbls.SecretKey

	err = secKey.Deserialize(validatorPrivkey.Marshal())
	if err != nil {
		return fmt.Errorf("cannot convert validator priv key")
	}

	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(common.DOMAIN_DEPOSIT, common.Version(genesis.GenesisForkVersion), common.Root{})
	msg := common.ComputeSigningRoot(msgRoot, dom)
	sig := secKey.SignHash(msg[:])
	copy(data.Signature[:], sig.Serialize())

	dataRoot := data.HashTreeRoot(tree.GetHashFn())

	// generate deposit transaction

	var client *execution.Client

	if t.config.ClientPattern == "" {
		client = clientPool.GetExecutionPool().GetReadyEndpoint(execution.UnspecifiedClient)
	} else {
		clients := clientPool.GetClientsByNamePatterns([]string{t.config.ClientPattern})
		if len(clients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}
		client = clients[0].ExecutionClient
	}

	depositContract, err := depositcontract.NewDepositContract(t.depositContractAddr, client.GetRPCClient().GetEthClient())
	if err != nil {
		return fmt.Errorf("cannot create bound instance of DepositContract: %w", err)
	}

	wallet, err := clientPool.GetExecutionPool().GetWalletByPrivkey(t.walletPrivKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	tx, err := wallet.BuildTransaction(ctx, func(ctx context.Context, nonce uint64, signer bind.SignerFn) (*ethtypes.Transaction, error) {
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
		return fmt.Errorf("cannot build deposit transaction: %w", err)
	}

	t.logger.Infof("sending deposit transaction (account idx: %v, nonce: %v)", accountIdx, tx.Nonce())

	err = client.GetRPCClient().SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed sending deposit transaction: %w", err)
	}

	go func() {
		if onConfirm != nil {
			defer onConfirm()
		}

		receipt, err := wallet.AwaitTransaction(ctx, tx)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			t.logger.Errorf("failed awaiting transaction receipt: %w", err)
			return
		}

		if receipt == nil {
			t.logger.Errorf("transaction replaced")
			return
		}
	}()

	return nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
