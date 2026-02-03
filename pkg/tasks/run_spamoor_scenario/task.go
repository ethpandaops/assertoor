package runspamoorscenario

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/spamoor/scenario"
	"github.com/ethpandaops/spamoor/scenarios"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var (
	TaskName       = "run_spamoor_scenario"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs a spamoor scenario with the given configuration",
		Category:    "transaction",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
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
	if valerr := config.Validate(); valerr != nil {
		return valerr
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	t.ctx.ReportProgress(0, "Initializing spamoor scenario...")

	// Get scenario descriptor
	scenarioDescriptor := scenarios.GetScenario(t.config.ScenarioName)
	if scenarioDescriptor == nil {
		return fmt.Errorf("scenario not found: %s", t.config.ScenarioName)
	}

	t.logger.Infof("running spamoor scenario: %s", t.config.ScenarioName)

	// Load root wallet
	privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	// Create wallet pool without preparation - scenario will configure it
	walletPool, err := walletMgr.NewWalletPoolByPrivkey(ctx, t.logger, privKey)
	if err != nil {
		return fmt.Errorf("failed to create wallet pool: %w", err)
	}

	rootWallet := walletPool.GetRootWallet().GetWallet()
	t.logger.Infof("root wallet: %v [nonce: %v] %v ETH",
		rootWallet.GetAddress().Hex(),
		rootWallet.GetNonce(),
		rootWallet.GetReadableBalance(18, 0, 4, false, false))

	// Create scenario instance
	scenarioInstance := scenarioDescriptor.NewScenario(t.logger)

	// Register flags (scenarios use pflags for configuration)
	flags := pflag.NewFlagSet(t.config.ScenarioName, pflag.ContinueOnError)

	err = scenarioInstance.Flags(flags)
	if err != nil {
		return fmt.Errorf("failed to register scenario flags: %w", err)
	}

	// Prepare scenario config YAML
	configYAML := ""

	if t.config.ScenarioYAML != nil {
		var configMap map[string]interface{}

		err = t.config.ScenarioYAML.Unmarshal(&configMap)
		if err != nil {
			return fmt.Errorf("failed to unmarshal scenario config: %w", err)
		}

		configBytes, marshalErr := yaml.Marshal(configMap)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal scenario config to YAML: %w", marshalErr)
		}

		configYAML = string(configBytes)
	}

	// Initialize scenario with options
	// The scenario will set wallet count and other settings in Init()
	scenarioOptions := &scenario.Options{
		WalletPool: walletPool,
		Config:     configYAML,
		GlobalCfg:  nil,
	}

	t.ctx.ReportProgress(10, "Initializing scenario...")

	err = scenarioInstance.Init(scenarioOptions)
	if err != nil {
		return fmt.Errorf("failed to initialize scenario: %w", err)
	}

	// Load wallet pool config from the scenario YAML (seed, refill settings, etc)
	if configYAML != "" {
		err = walletPool.LoadConfig(configYAML)
		if err != nil {
			return fmt.Errorf("failed to load wallet pool config: %w", err)
		}
	}

	t.ctx.ReportProgress(20, "Preparing wallets...")

	// Now prepare the wallets (create and fund them)
	err = walletPool.PrepareWallets()
	if err != nil {
		return fmt.Errorf("failed to prepare wallets: %w", err)
	}

	// Log wallet info after preparation
	for idx, wallet := range walletPool.GetAllWallets() {
		t.logger.Debugf("wallet #%v: %v [nonce: %v] %v ETH",
			idx,
			wallet.GetAddress().Hex(),
			wallet.GetNonce(),
			wallet.GetReadableBalance(18, 0, 4, false, false))
	}

	t.ctx.ReportProgress(30, "Running scenario...")

	// Run the scenario
	err = scenarioInstance.Run(ctx)
	if err != nil {
		return fmt.Errorf("scenario execution failed: %w", err)
	}

	if ctx.Err() != nil {
		t.ctx.ReportProgress(100, "Scenario completed")
		t.ctx.SetResult(types.TaskResultSuccess)
	} else {
		t.ctx.ReportProgress(100, "Scenario cancelled")
		t.ctx.SetResult(types.TaskResultFailure)
	}

	return nil
}
