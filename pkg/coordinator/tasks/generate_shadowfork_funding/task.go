package generateshadowforkfunding

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	shadowvaultcontract "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_shadowfork_funding/shadowvault_contract"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_shadowfork_funding"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates a transaction that requests funds from the ShadowForkVault on shadow forks.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

const shadowWithdrawGasLimit = 50000

type Task struct {
	ctx       *types.TaskContext
	options   *types.TaskOptions
	config    Config
	logger    logrus.FieldLogger
	wallet    *wallet.Wallet
	vaultAddr common.Address
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Name() string {
	return TaskDescriptor.Name
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
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
	if err2 := config.Validate(); err2 != nil {
		return err2
	}

	// load wallets
	privKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return err
	}

	t.wallet, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	// parse vault addr
	if config.ShadowForkVaultContract != "" {
		err = t.vaultAddr.UnmarshalText([]byte(config.ShadowForkVaultContract))
		if err != nil {
			return fmt.Errorf("cannot decode execution addr: %w", err)
		}
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	err := t.wallet.AwaitReady(ctx)
	if err != nil {
		return err
	}

	t.logger.Infof("root wallet: %v [nonce: %v]  %v ETH", t.wallet.GetAddress().Hex(), t.wallet.GetNonce(), t.wallet.GetReadableBalance(18, 0, 4, false, false))

	if t.wallet.GetBalance().Cmp(t.config.MinBalance) <= 0 {
		t.logger.Infof("balance exceeds minBalance (%v ETH), skipping shadow vault request", wallet.GetReadableBalance(t.config.MinBalance, 18, 0, 4, false, false))
		return nil
	}

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	client := clientPool.GetExecutionPool().AwaitReadyEndpoint(ctx, execution.AnyClient)
	if client == nil {
		return ctx.Err()
	}

	vaultContract, err := shadowvaultcontract.NewShadowvaultcontract(t.vaultAddr, client.GetRPCClient().GetEthClient())
	if err != nil {
		return fmt.Errorf("cannot create bound instance of ShadowVaultContract: %w", err)
	}

	// check shadow fork eligibility
	vaultGenesisTimeBigInt, err := vaultContract.GetGenesisTime(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return fmt.Errorf("error requesting genesis time from ShadowVaultContract: %w", err)
	}

	vaultGenesisTime := vaultGenesisTimeBigInt.Int64()

	consensusPool := clientPool.GetConsensusPool()
	consensusPool.AwaitReadyEndpoint(ctx, consensus.AnyClient)
	consensusGenesis := consensusPool.GetBlockCache().GetGenesis()

	if vaultGenesisTime >= consensusGenesis.GenesisTime.Unix() {
		return fmt.Errorf("cannot request funds from ShadowForkVault when not on a shadow fork")
	}

	wallclockSubscription := consensusPool.GetBlockCache().SubscribeWallclockSlotEvent(1)
	defer wallclockSubscription.Unsubscribe()

	// generate shadow withdrawal tx
	var withdrawalTx *ethtypes.Transaction

	minFeeBalance := big.NewInt(shadowWithdrawGasLimit)
	minFeeBalance = minFeeBalance.Mul(minFeeBalance, t.config.TxFeeCap)

	retry := 0

	for {
		retry++
		if retry > 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-wallclockSubscription.Channel():
			}
		}

		client := clientPool.GetExecutionPool().AwaitReadyEndpoint(ctx, execution.AnyClient)
		if client == nil {
			continue
		}

		balance, err := client.GetRPCClient().GetBalanceAt(ctx, t.wallet.GetAddress(), nil)
		if err != nil {
			continue
		}

		if balance.Cmp(minFeeBalance) < 0 {
			t.logger.Infof("not enough funds to pay withdrawal tx fee (have %v, need: %v)", wallet.GetReadableBalance(balance, 18, 0, 4, false, false), wallet.GetReadableBalance(minFeeBalance, 18, 0, 4, false, false))
			continue
		}

		if withdrawalTx == nil {
			tx, err2 := t.generateShadowWithdrawTx(ctx)
			if err2 != nil {
				t.logger.Infof("could not generate withdrawal: %v", err2)
				continue
			}

			withdrawalTx = tx
		}

		err = client.GetRPCClient().SendTransaction(ctx, withdrawalTx)
		if err != nil {
			t.logger.Infof("failed sending shadow withdraw transaction: %w", err)
			continue
		}

		break
	}

	if withdrawalTx != nil {
		// await confirmation
		receipt, err := t.wallet.AwaitTransaction(ctx, withdrawalTx)

		if err != nil {
			return err
		}

		t.logger.Infof("shadow withdraw tx confirmed! status: %v", receipt.Status)
		t.wallet.ResyncState()
	}

	return ctx.Err()
}

func (t *Task) generateShadowWithdrawTx(ctx context.Context) (*ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	consensusPool := clientPool.GetConsensusPool()
	consensusGenesis := consensusPool.GetBlockCache().GetGenesis()

	if consensusGenesis.GenesisTime.Compare(time.Now()) >= 0 {
		return nil, fmt.Errorf("before genesis time")
	}

	currentSlot, _, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return nil, fmt.Errorf("error reading wall clock: %v", err)
	}

	if currentSlot.Number() < 5 {
		return nil, fmt.Errorf("before slot 5 (slot: %v)", currentSlot.Number())
	}

	spec := consensusPool.GetBlockCache().GetSpecs()
	if spec == nil {
		return nil, fmt.Errorf("could not get spec")
	}

	// generate header proof for slot N-1
	elClient := clientPool.GetExecutionPool().AwaitReadyEndpoint(ctx, execution.AnyClient)
	if elClient == nil {
		return nil, ctx.Err()
	}

	clClient := clientPool.GetAllClients()[elClient.GetIndex()].ConsensusClient
	_, clHeadRoot := clClient.GetLastHead()

	clBlock := consensusPool.GetBlockCache().GetCachedBlockByRoot(clHeadRoot)
	if clBlock == nil {
		return nil, fmt.Errorf("could not get head block")
	}

	clParentRoot := clBlock.GetParentRoot()
	if clParentRoot == nil {
		return nil, fmt.Errorf("could not get parent root")
	}

	clBlock = consensusPool.GetBlockCache().GetCachedBlockByRoot(*clParentRoot)
	if clBlock == nil {
		return nil, fmt.Errorf("could not get parent block")
	}

	clBlockHead := clBlock.AwaitHeader(ctx, 1*time.Second)
	if clBlockHead == nil {
		return nil, fmt.Errorf("could not get parent block header")
	}

	vaultContract, err := shadowvaultcontract.NewShadowvaultcontract(t.vaultAddr, elClient.GetRPCClient().GetEthClient())
	if err != nil {
		return nil, fmt.Errorf("cannot create bound instance of ShadowVaultContract: %w", err)
	}

	proof, err := vaultContract.GenerateHeaderProof(
		&bind.CallOpts{
			Context: ctx,
		},
		big.NewInt(int64(clBlockHead.Message.Slot)),
		big.NewInt(int64(clBlockHead.Message.ProposerIndex)),
		clBlockHead.Message.ParentRoot,
		clBlockHead.Message.StateRoot,
		clBlockHead.Message.BodyRoot,
		big.NewInt(0),
	)
	if err != nil {
		return nil, fmt.Errorf("error while creating header proof: %w", err)
	}

	// create request transaction
	tx, err := t.wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, signer bind.SignerFn) (*ethtypes.Transaction, error) {
		return vaultContract.ShadowWithdraw(
			&bind.TransactOpts{
				From:      t.wallet.GetAddress(),
				Nonce:     big.NewInt(int64(nonce)),
				Value:     big.NewInt(0),
				GasLimit:  50000,
				GasFeeCap: t.config.TxFeeCap,
				GasTipCap: t.config.TxTipCap,
				Signer:    signer,
				NoSend:    true,
			},
			big.NewInt(consensusGenesis.GenesisTime.Unix()+(int64(clBlockHead.Message.Slot)*int64(spec.SecondsPerSlot))),
			big.NewInt(int64(clBlockHead.Message.Slot)),
			proof,
			t.wallet.GetAddress(),
			t.config.RequestAmount,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build shadow withdraw transaction: %w", err)
	}

	return tx, nil
}
