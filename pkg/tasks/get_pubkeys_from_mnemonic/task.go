package getpubkeysfrommnemonic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "get_pubkeys_from_mnemonic"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Get public keys from mnemonic",
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
	mnemonic := strings.TrimSpace(t.config.Mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return errors.New("mnemonic is not valid")
	}

	seed := bip39.NewSeed(mnemonic, "")
	pubkeys := make([]string, 0, t.config.Count)

	for index := t.config.StartIndex; index < t.config.StartIndex+t.config.Count; index++ {
		validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", index)

		validatorPriv, err := util.PrivateKeyFromSeedAndPath(seed, validatorKeyPath)
		if err != nil {
			return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
		}

		validatorPubkey := validatorPriv.PublicKey().Marshal()
		pubkey := fmt.Sprintf("0x%x", validatorPubkey)

		t.logger.Infof("Generated pubkey %v: %v", index, pubkey)

		pubkeys = append(pubkeys, pubkey)
	}

	t.ctx.Outputs.SetVar("pubkeys", pubkeys)

	return nil
}
