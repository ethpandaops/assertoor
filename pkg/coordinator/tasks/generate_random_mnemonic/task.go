package generaterandommnemonic

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
)

var (
	TaskName       = "generate_random_mnemonic"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generate random mnemonic.",
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

func (t *Task) Execute(_ context.Context) error {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return fmt.Errorf("could not create entropy: %v", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return fmt.Errorf("could not create mnemonic: %v", err)
	}

	t.logger.Infof("Generated random mnemonic: %v", mnemonic)

	if t.config.MnemonicResultVar != "" {
		t.ctx.Vars.SetVar(t.config.MnemonicResultVar, mnemonic)
	}

	return nil
}
