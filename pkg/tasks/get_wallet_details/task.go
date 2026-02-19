package getwalletdetails

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "get_wallet_details"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Get wallet details.",
		Category:    "wallet",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "address",
				Type:        "string",
				Description: "The wallet address.",
			},
			{
				Name:        "balance",
				Type:        "string",
				Description: "The wallet balance in wei as a string.",
			},
			{
				Name:        "nonce",
				Type:        "uint64",
				Description: "The current nonce of the wallet.",
			},
			{
				Name:        "summary",
				Type:        "object",
				Description: "Summary object with wallet details.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

type Summary struct {
	Address        string `json:"address"`
	PrivKey        string `json:"privKey"`
	PendingNonce   uint64 `json:"pendingNonce"`
	ConfirmedNonce uint64 `json:"confirmedNonce"`
	Balance        string `json:"balance"`
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

func (t *Task) Execute(_ context.Context) error {
	var wal *spamoor.Wallet

	if t.config.PrivateKey != "" {
		privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
		if err != nil {
			return err
		}

		wal, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.ctx.Scheduler.GetTestRunCtx(), privKey)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet: %w", err)
		}
	} else {
		address := common.HexToAddress(t.config.Address)

		var err error

		wal, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByAddress(t.ctx.Scheduler.GetTestRunCtx(), address)
		if err != nil {
			return fmt.Errorf("cannot initialize wallet: %w", err)
		}
	}

	t.ctx.Outputs.SetVar("address", wal.GetAddress())
	t.ctx.Outputs.SetVar("balance", wal.GetBalance().String())
	t.ctx.Outputs.SetVar("nonce", wal.GetNonce())
	t.ctx.Outputs.SetVar("summary", &Summary{
		Address:        wal.GetAddress().String(),
		PrivKey:        fmt.Sprintf("%x", crypto.FromECDSA(wal.GetPrivateKey())),
		PendingNonce:   wal.GetNonce(),
		ConfirmedNonce: wal.GetConfirmedNonce(),
		Balance:        wal.GetBalance().String(),
	})

	t.ctx.ReportProgress(100, "Wallet details retrieved")

	return nil
}
