package generatechildwallet

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_child_wallet"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates a funded child wallet.",
		Category:    "wallet",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "childWallet",
				Type:        "object",
				Description: "Summary of the generated child wallet including address and balance.",
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
	PrivKey        string `json:"privkey"`
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
	if err2 := config.Validate(); err2 != nil {
		return err2
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	t.ctx.ReportProgress(0, "Preparing root wallet...")

	// Load root wallet
	privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
	if err != nil {
		return err
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	// Use test run context for wallet operations so they don't get cancelled when this task completes
	testRunCtx := t.ctx.Scheduler.GetTestRunCtx()

	rootWallet, err := walletMgr.GetWalletByPrivkey(testRunCtx, privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	t.logger.Infof("root wallet: %v [nonce: %v]  %v ETH", rootWallet.GetAddress().Hex(), rootWallet.GetNonce(), rootWallet.GetReadableBalance(18, 0, 4, false, false))
	t.ctx.ReportProgress(25, "Generating child wallet...")

	walletSeed := t.config.WalletSeed
	if t.config.RandomSeed {
		walletSeed = t.randStringBytes(20)
	}

	walletPool, err := walletMgr.GetWalletPoolByPrivkey(testRunCtx, t.logger, privKey, &txmgr.WalletPoolConfig{
		WalletCount:   1,
		WalletSeed:    walletSeed,
		RefillAmount:  uint256.MustFromBig(t.config.PrefundAmount),
		RefillBalance: uint256.MustFromBig(t.config.PrefundMinBalance),
	})
	if err != nil {
		return err
	}

	t.ctx.ReportProgress(75, "Child wallet funded...")

	// Stop the funding loop if keepFunding is not enabled
	if !t.config.KeepFunding {
		walletPool.StopFunding()
	}

	childWallet := walletPool.GetWallet(spamoor.SelectWalletByIndex, 0)
	t.logger.Infof("child wallet: %v [nonce: %v]  %v ETH", childWallet.GetAddress().Hex(), childWallet.GetNonce(), childWallet.GetReadableBalance(18, 0, 4, false, false))

	walletSummary := &Summary{
		Address:        childWallet.GetAddress().String(),
		PrivKey:        fmt.Sprintf("%x", crypto.FromECDSA(childWallet.GetPrivateKey())),
		PendingNonce:   childWallet.GetNonce(),
		ConfirmedNonce: childWallet.GetConfirmedNonce(),
		Balance:        childWallet.GetBalance().String(),
	}

	if walletSummaryData, err := vars.GeneralizeData(walletSummary); err == nil {
		t.ctx.Outputs.SetVar("childWallet", walletSummaryData)
	} else {
		t.logger.Warnf("Failed setting `childWallet` output: %v", err)
	}

	if t.config.WalletAddressResultVar != "" {
		t.ctx.Vars.SetVar(t.config.WalletAddressResultVar, childWallet.GetAddress().Hex())
	}

	if t.config.WalletPrivateKeyResultVar != "" {
		t.ctx.Vars.SetVar(t.config.WalletPrivateKeyResultVar, fmt.Sprintf("%x", crypto.FromECDSA(childWallet.GetPrivateKey())))
	}

	t.ctx.ReportProgress(100, "Child wallet generated")

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
