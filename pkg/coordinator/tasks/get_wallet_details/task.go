package getwalletdetails

import (
	"context"
	"fmt"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "get_wallet_details"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Get wallet details.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
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
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	var wal *wallet.Wallet

	if t.config.PrivateKey != "" {
		privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
		if err != nil {
			return err
		}

		wal, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet: %w", err)
		}
	} else {
		address := common.HexToAddress(t.config.Address)
		wal = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByAddress(address)
	}

	err := wal.AwaitReady(ctx)
	if err != nil {
		return err
	}

	t.ctx.Outputs.SetVar("address", wal.GetAddress())
	t.ctx.Outputs.SetVar("balance", wal.GetBalance().String())
	t.ctx.Outputs.SetVar("nonce", wal.GetNonce())
	t.ctx.Outputs.SetVar("summary", wal.GetSummary())

	return nil
}
