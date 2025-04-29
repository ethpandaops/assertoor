package generateslashings

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	ethcommon "github.com/ethereum/go-ethereum/common"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/noku-team/assertoor/pkg/coordinator/clients/consensus"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	e2types "github.com/wealdtech/go-eth2-types/v2"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_slashings"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates slashable attestations / proposals and sends them to the network",
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

	t.withdrSeed, err = t.mnemonicToSeed(config.Mnemonic)
	if err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.config.StartIndex > 0 {
		t.nextIndex = uint64(t.config.StartIndex) //nolint:gosec // no overflow possible
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount) //nolint:gosec // no overflow possible
	}

	var subscription *consensus.Subscription[*consensus.Block]
	if t.config.LimitPerSlot > 0 {
		subscription = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	forkState, err := t.loadChainState(ctx)
	if err != nil {
		return err
	}

	perSlotCount := 0
	totalCount := 0

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		err := t.generateSlashing(ctx, accountIdx, forkState)
		if err != nil {
			t.logger.Errorf("error generating slashing: %v", err.Error())
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

func (t *Task) loadChainState(ctx context.Context) (*phase0.Fork, error) {
	client := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().AwaitReadyEndpoint(ctx, consensus.AnyClient)
	if client == nil {
		return nil, ctx.Err()
	}

	forkState, err := client.GetRPCClient().GetForkState(ctx, "head")
	if err != nil {
		return nil, err
	}

	return forkState, nil
}

func (t *Task) generateSlashing(ctx context.Context, accountIdx uint64, forkState *phase0.Fork) error {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, validatorKeyPath)
	if err != nil {
		return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()
	for _, val := range t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet() {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if validator == nil {
		return fmt.Errorf("validator not found")
	}

	if validator.Status != v1.ValidatorStateActiveOngoing {
		return fmt.Errorf("validator %v is not active", validator.Index)
	}

	var attesterSlashing *phase0.AttesterSlashing

	var proposerSlashing *phase0.ProposerSlashing

	switch t.config.SlashingType {
	case "attester", "surround_attester":
		attesterSlashing, err = t.generateSurroundAttesterSlashing(uint64(validator.Index), validatorPrivkey, forkState)
	case "proposer":
		proposerSlashing, err = t.generateProposerSlashing(uint64(validator.Index), validatorPrivkey, forkState)
	default:
		return fmt.Errorf("unknown slashing type: %v", t.config.SlashingType)
	}

	if err != nil {
		return fmt.Errorf("failed generating slashing: %v", err)
	}

	if attesterSlashing == nil && proposerSlashing == nil {
		return fmt.Errorf("no slashing generated")
	}

	var client *consensus.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetConsensusPool().GetReadyEndpoint(consensus.AnyClient)
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		client = clients[0].ConsensusClient
	}

	if attesterSlashing != nil {
		t.logger.WithField("client", client.GetName()).Infof("sending attester slashing for validator %v", validator.Index)

		err = client.GetRPCClient().SubmitAttesterSlashing(ctx, attesterSlashing)
		if err != nil {
			return err
		}
	}

	if proposerSlashing != nil {
		t.logger.WithField("client", client.GetName()).Infof("sending proposer slashing for validator %v", validator.Index)

		err = client.GetRPCClient().SubmitProposerSlashing(ctx, proposerSlashing)
		if err != nil {
			return err
		}
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

func (t *Task) generateSurroundAttesterSlashing(validatorIndex uint64, validatorKey *e2types.BLSPrivateKey, forkState *phase0.Fork) (*phase0.AttesterSlashing, error) {
	// surround attester slashing case:
	// different target, different source
	// source1 < source 2
	// target 1 > target 2
	clPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	slot, epoch, _ := clPool.GetBlockCache().GetWallclock().Now()
	if epoch.Number() < 4 {
		return nil, fmt.Errorf("current epoch too low (require epoch >= 4)")
	}

	specs := clPool.GetBlockCache().GetSpecs()
	genesis := clPool.GetBlockCache().GetGenesis()

	slot1 := slot.Number()
	slot2 := slot1 - specs.SlotsPerEpoch - 2

	targetEpoch1 := slot1 / specs.SlotsPerEpoch
	sourceEpoch1 := targetEpoch1 - 3

	targetEpoch2 := slot2 / specs.SlotsPerEpoch
	sourceEpoch2 := targetEpoch2

	source1 := &phase0.Checkpoint{
		Epoch: phase0.Epoch(sourceEpoch1),
		Root:  phase0.Root(ethcommon.FromHex("0x1010101010101010101010101010101010101010101010101010101010101010")),
	}
	target1 := &phase0.Checkpoint{
		Epoch: phase0.Epoch(targetEpoch1),
		Root:  phase0.Root(ethcommon.FromHex("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")),
	}

	source2 := &phase0.Checkpoint{
		Epoch: phase0.Epoch(sourceEpoch2),
		Root:  phase0.Root(ethcommon.FromHex("0x1010101010101010101010101010101010101010101010101010101010101010")),
	}
	target2 := &phase0.Checkpoint{
		Epoch: phase0.Epoch(targetEpoch2),
		Root:  phase0.Root(ethcommon.FromHex("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")),
	}

	committeeIndex := uint64(0)
	dom := common.ComputeDomain(common.DOMAIN_BEACON_ATTESTER, common.Version(forkState.CurrentVersion), tree.Root(genesis.GenesisValidatorsRoot))

	var secKey hbls.SecretKey

	err := secKey.Deserialize(validatorKey.Marshal())
	if err != nil {
		return nil, fmt.Errorf("failed converting validator priv key: %w", err)
	}

	attestationData1 := &phase0.AttestationData{
		Slot:            phase0.Slot(slot1),
		Index:           phase0.CommitteeIndex(committeeIndex),
		BeaconBlockRoot: phase0.Root(ethcommon.FromHex("0x00000000219ab540356cBB839Cbe05303d7705Fa424242424242424242424242")),
		Source:          source1,
		Target:          target1,
	}
	attestationData2 := &phase0.AttestationData{
		Slot:            phase0.Slot(slot2),
		Index:           phase0.CommitteeIndex(committeeIndex),
		BeaconBlockRoot: phase0.Root(ethcommon.FromHex("0x00000000219ab540356cBB839Cbe05303d7705Fa424242424242424242424242")),
		Source:          source2,
		Target:          target2,
	}

	msgRoot1, err := attestationData1.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot build attestation1 data tree root: %v", err)
	}

	signingRoot1 := common.ComputeSigningRoot(msgRoot1, dom)
	sig1 := secKey.SignHash(signingRoot1[:])

	msgRoot2, err := attestationData2.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot build attestation2 data tree root: %v", err)
	}

	signingRoot2 := common.ComputeSigningRoot(msgRoot2, dom)
	sig2 := secKey.SignHash(signingRoot2[:])

	att1 := &phase0.IndexedAttestation{
		AttestingIndices: []uint64{validatorIndex},
		Data:             attestationData1,
		Signature:        phase0.BLSSignature(sig1.Serialize()),
	}
	att2 := &phase0.IndexedAttestation{
		AttestingIndices: []uint64{validatorIndex},
		Data:             attestationData2,
		Signature:        phase0.BLSSignature(sig2.Serialize()),
	}

	return &phase0.AttesterSlashing{
		Attestation1: att1,
		Attestation2: att2,
	}, nil
}

func (t *Task) generateProposerSlashing(validatorIndex uint64, validatorKey *e2types.BLSPrivateKey, forkState *phase0.Fork) (*phase0.ProposerSlashing, error) {
	clPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	genesis := clPool.GetBlockCache().GetGenesis()

	slot, _, _ := clPool.GetBlockCache().GetWallclock().Now()

	headerData1 := &phase0.BeaconBlockHeader{
		Slot:          phase0.Slot(slot.Number()),
		ProposerIndex: phase0.ValidatorIndex(validatorIndex),
		ParentRoot:    phase0.Root(ethcommon.FromHex("0x1010101010101010101010101010101010101010101010101010101010101010")),
		StateRoot:     phase0.Root(ethcommon.FromHex("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")),
		BodyRoot:      phase0.Root(ethcommon.FromHex("0xa1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1")),
	}
	headerData2 := &phase0.BeaconBlockHeader{
		Slot:          phase0.Slot(slot.Number()),
		ProposerIndex: phase0.ValidatorIndex(validatorIndex),
		ParentRoot:    phase0.Root(ethcommon.FromHex("0x1010101010101010101010101010101010101010101010101010101010101010")),
		StateRoot:     phase0.Root(ethcommon.FromHex("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")),
		BodyRoot:      phase0.Root(ethcommon.FromHex("0xb1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1b1")),
	}

	dom := common.ComputeDomain(common.DOMAIN_BEACON_PROPOSER, common.Version(forkState.CurrentVersion), tree.Root(genesis.GenesisValidatorsRoot))

	var secKey hbls.SecretKey

	err := secKey.Deserialize(validatorKey.Marshal())
	if err != nil {
		return nil, fmt.Errorf("failed converting validator priv key: %w", err)
	}

	msgRoot1, err := headerData1.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot build header1 data tree root: %v", err)
	}

	signingRoot1 := common.ComputeSigningRoot(msgRoot1, dom)
	sig1 := secKey.SignHash(signingRoot1[:])

	msgRoot2, err := headerData2.HashTreeRoot()
	if err != nil {
		return nil, fmt.Errorf("cannot build header2 data tree root: %v", err)
	}

	signingRoot2 := common.ComputeSigningRoot(msgRoot2, dom)
	sig2 := secKey.SignHash(signingRoot2[:])

	header1 := &phase0.SignedBeaconBlockHeader{
		Message:   headerData1,
		Signature: phase0.BLSSignature(sig1.Serialize()),
	}
	header2 := &phase0.SignedBeaconBlockHeader{
		Message:   headerData2,
		Signature: phase0.BLSSignature(sig2.Serialize()),
	}

	return &phase0.ProposerSlashing{
		SignedHeader1: header1,
		SignedHeader2: header2,
	}, nil
}
