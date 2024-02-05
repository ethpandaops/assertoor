package generateexits

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_exits"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates voluntary exits and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	withdrSeed []byte
	nextIndex  uint64
	lastIndex  uint64
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

	t.withdrSeed, err = t.mnemonicToSeed(config.Mnemonic)
	if err != nil {
		return err
	}

	t.config = config

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

	fork, validators, err := t.loadChainState(ctx)
	if err != nil {
		return err
	}

	perSlotCount := 0
	totalCount := 0

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		err := t.generateVoluntaryExit(ctx, accountIdx, fork, validators)
		if err != nil {
			t.logger.Errorf("error generating voluntary exit: %v", err.Error())
		} else {
			t.ctx.SetResult(types.TaskResultSuccess)
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

func (t *Task) loadChainState(ctx context.Context) (*phase0.Fork, map[phase0.ValidatorIndex]*v1.Validator, error) {
	client := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)

	fork, err := client.GetRPCClient().GetForkState(ctx, "head")
	if err != nil {
		return nil, nil, err
	}

	validators, err := client.GetRPCClient().GetStateValidators(ctx, "head")
	if err != nil {
		return nil, nil, err
	}

	return fork, validators, nil
}

func (t *Task) generateVoluntaryExit(ctx context.Context, accountIdx uint64, fork *phase0.Fork, validators map[phase0.ValidatorIndex]*v1.Validator) error {
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, validatorKeyPath)
	if err != nil {
		return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	// select validator
	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()
	for _, val := range validators {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	// check validator status
	if validator == nil {
		return fmt.Errorf("validator not found")
	}

	if validator.Validator.ExitEpoch != 18446744073709551615 {
		return fmt.Errorf("validator %v is already exited", validator.Index)
	}

	// select client
	var client *consensus.Client

	clientPool := t.ctx.Scheduler.GetCoordinator().ClientPool()
	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}
		client = clients[0].ConsensusClient
	}

	// build voluntary exit message
	currentSlot, _ := client.GetLastHead()
	specs := clientPool.GetConsensusPool().GetBlockCache().GetSpecs()
	currentEpoch := phase0.Epoch(currentSlot / phase0.Slot(specs.SlotsPerEpoch))
	operation := &phase0.VoluntaryExit{
		Epoch:          currentEpoch,
		ValidatorIndex: validator.Index,
	}

	root, err := operation.HashTreeRoot()
	if err != nil {
		return fmt.Errorf("failed to generate root for exit operation: %w", err)
	}

	var secKey hbls.SecretKey

	err = secKey.Deserialize(validatorPrivkey.Marshal())
	if err != nil {
		return fmt.Errorf("failed converting validator priv key: %w", err)
	}

	forkVersion := fork.CurrentVersion
	if uint64(fork.Epoch) >= specs.CappellaForkEpoch {
		forkVersion = specs.CappellaForkVersion
	}

	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(common.DOMAIN_VOLUNTARY_EXIT, common.Version(forkVersion), tree.Root(genesis.GenesisValidatorsRoot))
	signingRoot := common.ComputeSigningRoot(root, dom)
	sig := secKey.SignHash(signingRoot[:])

	var signedMsg phase0.SignedVoluntaryExit

	signedMsg.Message = operation
	copy(signedMsg.Signature[:], sig.Serialize())

	t.logger.WithField("client", client.GetName()).Infof("sending voluntary exit for validator %v", validator.Index)

	err = client.GetRPCClient().SubmitVoluntaryExits(ctx, &signedMsg)
	if err != nil {
		return err
	}

	return nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
