package generateblschanges

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/capella"
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
	TaskName       = "generate_bls_changes"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates bls changes and sends them to the network",
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
	targetAddr common.Eth1Address
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Scheduler.GetLogger().WithField("task", TaskName),
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

	err = t.targetAddr.UnmarshalText([]byte(config.TargetAddress))
	if err != nil {
		return fmt.Errorf("cannot decode execution addr: %w", err)
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

	genesis, validators, err := t.loadChainState(ctx)
	if err != nil {
		return err
	}

	perSlotCount := 0
	totalCount := 0

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		err := t.generateBlsChange(ctx, accountIdx, genesis, validators)
		if err != nil {
			t.logger.Errorf("error generating bls change: %v", err.Error())
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

func (t *Task) loadChainState(ctx context.Context) (*v1.Genesis, map[phase0.ValidatorIndex]*v1.Validator, error) {
	client := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)

	genesis, err := client.GetRPCClient().GetGenesis(ctx)
	if err != nil {
		return nil, nil, err
	}

	validators, err := client.GetRPCClient().GetStateValidators(ctx, "head")
	if err != nil {
		return nil, nil, err
	}

	return genesis, validators, nil
}

func (t *Task) generateBlsChange(ctx context.Context, accountIdx uint64, genesis *v1.Genesis, validators map[phase0.ValidatorIndex]*v1.Validator) error {
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, validatorKeyPath)
	if err != nil {
		return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()
	for _, val := range validators {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if validator == nil {
		return fmt.Errorf("validator not found")
	}

	if validator.Validator.WithdrawalCredentials[0] != 0x00 {
		return fmt.Errorf("validator does not have 0x00 withdrawal creds")
	}

	withdrAccPath := fmt.Sprintf("m/12381/3600/%d/0", accountIdx)

	withdr, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, withdrAccPath)
	if err != nil {
		return fmt.Errorf("failed generating key %v: %w", withdrAccPath, err)
	}

	var withdrPub common.BLSPubkey

	copy(withdrPub[:], withdr.PublicKey().Marshal())

	msg := common.BLSToExecutionChange{
		ValidatorIndex:     common.ValidatorIndex(validator.Index),
		FromBLSPubKey:      withdrPub,
		ToExecutionAddress: t.targetAddr,
	}

	var secKey hbls.SecretKey

	err = secKey.Deserialize(withdr.Marshal())
	if err != nil {
		return fmt.Errorf("failed converting validator priv key: %w", err)
	}

	msgRoot := msg.HashTreeRoot(tree.GetHashFn())
	dom := common.ComputeDomain(common.DOMAIN_BLS_TO_EXECUTION_CHANGE, common.Version(genesis.GenesisForkVersion), tree.Root(genesis.GenesisValidatorsRoot))
	signingRoot := common.ComputeSigningRoot(msgRoot, dom)
	sig := secKey.SignHash(signingRoot[:])

	var signedMsg common.SignedBLSToExecutionChange

	signedMsg.BLSToExecutionChange = msg
	copy(signedMsg.Signature[:], sig.Serialize())

	signedChangeJSON, err := json.Marshal(&signedMsg)
	if err != nil {
		return err
	}

	var client *consensus.Client

	clientPool := t.ctx.Scheduler.GetCoordinator().ClientPool()
	if t.config.ClientPattern == "" {
		client = clientPool.GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)
	} else {
		clients := clientPool.GetClientsByNamePatterns([]string{t.config.ClientPattern})
		if len(clients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}
		client = clients[0].ConsensusClient
	}

	blsChange := &capella.SignedBLSToExecutionChange{}

	err = json.Unmarshal(signedChangeJSON, blsChange)
	if err != nil {
		return err
	}

	t.logger.WithField("client", client.GetName()).Infof("sending bls change for validator %v", validator.Index)

	err = client.GetRPCClient().SubmitBLSToExecutionChanges(ctx, []*capella.SignedBLSToExecutionChange{blsChange})
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
