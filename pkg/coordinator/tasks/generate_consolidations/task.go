package generateconsolidations

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/electra"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	e2types "github.com/wealdtech/go-eth2-types/v2"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_consolidations"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates consolidations and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

var DOMAIN_CONSOLIDATION = common.BLSDomainType{0x0B, 0x00, 0x00, 0x00}

type Task struct {
	ctx           *types.TaskContext
	options       *types.TaskOptions
	config        Config
	logger        logrus.FieldLogger
	sourceSeed    []byte
	targetSeed    []byte
	nextIndex     uint64
	lastIndex     uint64
	targetPrivKey *e2types.BLSPrivateKey
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

	t.sourceSeed, err = t.mnemonicToSeed(config.SourceMnemonic)
	if err != nil {
		return err
	}

	t.targetSeed, err = t.mnemonicToSeed(config.TargetMnemonic)
	if err != nil {
		return err
	}

	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", config.TargetKeyIndex)
	t.targetPrivKey, err = util.PrivateKeyFromSeedAndPath(t.targetSeed, validatorKeyPath)
	if err != nil {
		return fmt.Errorf("failed generating target validator key %v: %w", validatorKeyPath, err)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.config.SourceStartIndex > 0 {
		t.nextIndex = uint64(t.config.SourceStartIndex)
	}

	if t.config.SourceIndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.SourceIndexCount)
	}

	var subscription *consensus.Subscription[*consensus.Block]
	if t.config.LimitPerSlot > 0 {
		subscription = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	perSlotCount := 0
	totalCount := 0

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		err := t.generateConsolidation(ctx, accountIdx)
		if err != nil {
			t.logger.Errorf("error generating consolidationn: %v", err.Error())
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

func (t *Task) generateConsolidation(ctx context.Context, accountIdx uint64) error {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.sourceSeed, validatorKeyPath)
	if err != nil {
		return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	validatorSet := clientPool.GetConsensusPool().GetValidatorSet()
	specs := clientPool.GetConsensusPool().GetBlockCache().GetSpecs()

	var sourceValidator *v1.Validator
	var targetValidator *v1.Validator

	sourceValidatorPubkey := validatorPrivkey.PublicKey().Marshal()
	targetValidatorPubkey := t.targetPrivKey.PublicKey().Marshal()
	for _, val := range validatorSet {
		if bytes.Equal(val.Validator.PublicKey[:], sourceValidatorPubkey) {
			sourceValidator = val
		}
		if bytes.Equal(val.Validator.PublicKey[:], targetValidatorPubkey) {
			targetValidator = val
		}
	}

	if sourceValidator == nil {
		return fmt.Errorf("source validator not found")
	}
	if targetValidator == nil {
		return fmt.Errorf("source validator not found")
	}

	if sourceValidator.Validator.WithdrawalCredentials[0] != 0x01 {
		return fmt.Errorf("validator %v does not have 0x01 withdrawal creds", sourceValidator.Index)
	}
	if targetValidator.Validator.WithdrawalCredentials[0] != 0x01 {
		return fmt.Errorf("validator %v does not have 0x01 withdrawal creds", targetValidator.Index)
	}

	// select client
	var client *consensus.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetConsensusPool().AwaitReadyEndpoint(ctx, consensus.AnyClient)
		if client == nil {
			return ctx.Err()
		}
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		client = clients[0].ConsensusClient
	}

	// generate operation
	operation := electra.Consolidation{
		SourceIndex: sourceValidator.Index,
		TargetIndex: targetValidator.Index,
	}
	if t.config.ConsolidationEpoch >= 0 {
		operation.Epoch = phase0.Epoch(t.config.ConsolidationEpoch)
	} else {
		currentSlot, _ := client.GetLastHead()
		operation.Epoch = phase0.Epoch(currentSlot / phase0.Slot(specs.SlotsPerEpoch))
	}

	// sign operation
	operationRoot, err := operation.HashTreeRoot()
	if err != nil {
		return fmt.Errorf("could not generate hash tree root for consolidation operation: %v", err)
	}

	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(DOMAIN_CONSOLIDATION, common.Version(genesis.GenesisForkVersion), tree.Root(genesis.GenesisValidatorsRoot))
	signingRoot := common.ComputeSigningRoot(operationRoot, dom)

	// source signature
	var sourceSecKey hbls.SecretKey

	err = sourceSecKey.Deserialize(validatorPrivkey.Marshal())
	if err != nil {
		return fmt.Errorf("failed converting validator priv key: %w", err)
	}
	sig1 := sourceSecKey.SignHash(signingRoot[:])

	// target signature
	var targetSecKey hbls.SecretKey

	err = targetSecKey.Deserialize(t.targetPrivKey.Marshal())
	if err != nil {
		return fmt.Errorf("failed converting validator priv key: %w", err)
	}
	sig2 := targetSecKey.SignHash(signingRoot[:])

	// aggregate signature
	sig1.Add(sig2)

	// build signed operation
	operationSigned := electra.SignedConsolidation{
		Message:   &operation,
		Signature: phase0.BLSSignature(sig1.Serialize()),
	}

	t.logger.WithField("client", client.GetName()).Infof("sending consolidation for validator %v (consolidating to %v)", sourceValidator.Index, targetValidator.Index)

	err = client.GetRPCClient().SubmitConsolidation(ctx, &operationSigned)
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
