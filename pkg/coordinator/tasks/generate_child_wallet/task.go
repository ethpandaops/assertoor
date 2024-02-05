package generatechildwallet

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_child_wallet"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates a funded child wallet.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	wallet  *wallet.Wallet
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

	// Load root wallet
	privKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return err
	}

	t.wallet, err = t.ctx.Scheduler.GetCoordinator().WalletManager().GetWalletByPrivkey(privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
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

	walletSeed := t.config.WalletSeed
	if t.config.RandomSeed {
		walletSeed = t.randStringBytes(20)
	}

	walletPool, err := t.ctx.Scheduler.GetCoordinator().WalletManager().GetWalletPoolByPrivkey(t.wallet.GetPrivateKey(), 1, walletSeed)
	if err != nil {
		return err
	}

	err = walletPool.EnsureFunding(ctx, t.config.PrefundMinBalance, t.config.PrefundAmount, t.config.PrefundFeeCap, t.config.PrefundTipCap, 1)
	if err != nil {
		return err
	}

	childWallet := walletPool.GetNextChildWallet()
	t.logger.Infof("child wallet: %v [nonce: %v]  %v ETH", childWallet.GetAddress().Hex(), childWallet.GetNonce(), childWallet.GetReadableBalance(18, 0, 4, false, false))

	if t.config.WalletAddressResultVar != "" {
		t.ctx.Vars.SetVar(t.config.WalletAddressResultVar, childWallet.GetAddress().Hex())
	}

	if t.config.WalletPrivateKeyResultVar != "" {
		t.ctx.Vars.SetVar(t.config.WalletPrivateKeyResultVar, fmt.Sprintf("%x", crypto.FromECDSA(childWallet.GetPrivateKey())))
	}

	return ctx.Err()
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (t *Task) randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		//nolint:gosec // ignore
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(b)
}
