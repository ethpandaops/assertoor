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
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/types"
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
		Category:    "validator",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "exitedValidators",
				Type:        "array",
				Description: "Array of validator indices that were submitted for exit.",
			},
			{
				Name:        "includedExits",
				Type:        "number",
				Description: "Number of exits included on-chain (when awaitInclusion is enabled).",
			},
		},
		NewTask: NewTask,
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

	fork, err := t.loadChainState(ctx)
	if err != nil {
		return err
	}

	perSlotCount := 0
	totalCount := 0

	// Track submitted validator indices for awaitInclusion
	pendingValidators := make(map[phase0.ValidatorIndex]bool)
	exitedValidators := []uint64{}

	// Calculate target count for progress reporting
	targetCount := 0
	if t.config.LimitTotal > 0 {
		targetCount = t.config.LimitTotal
	} else if t.lastIndex > 0 {
		targetCount = int(t.lastIndex - t.nextIndex) //nolint:gosec // no overflow possible
	}

	t.ctx.ReportProgress(0, "Starting voluntary exit generation")

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		validatorIndex, err := t.generateVoluntaryExit(ctx, accountIdx, fork)
		if err != nil {
			t.logger.Errorf("error generating voluntary exit for index %v: %v", accountIdx, err.Error())
		} else {
			exitedValidators = append(exitedValidators, uint64(validatorIndex))
			t.ctx.Outputs.SetVar("exitedValidators", exitedValidators)

			if t.config.AwaitInclusion {
				pendingValidators[validatorIndex] = true
			} else {
				t.ctx.SetResult(types.TaskResultSuccess)
			}

			perSlotCount++
			totalCount++

			// Report progress
			if targetCount > 0 {
				progress := float64(totalCount) / float64(targetCount) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d voluntary exits", totalCount, targetCount))
			} else {
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d voluntary exits", totalCount))
			}
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

	if totalCount == 0 {
		t.ctx.SetResult(types.TaskResultFailure)

		return nil
	}

	// Await inclusion in blocks if configured
	if t.config.AwaitInclusion && len(pendingValidators) > 0 {
		err := t.awaitInclusion(ctx, pendingValidators, totalCount)
		if err != nil {
			return err
		}
	} else {
		t.ctx.ReportProgress(100, fmt.Sprintf("Completed generating %d voluntary exits", totalCount))
	}

	return nil
}

func (t *Task) awaitInclusion(ctx context.Context, pendingValidators map[phase0.ValidatorIndex]bool, totalCount int) error {
	blockSubscription := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	includedCount := 0
	t.ctx.Outputs.SetVar("includedExits", includedCount)

	t.logger.Infof("waiting for %d voluntary exits to be included in blocks", len(pendingValidators))
	t.ctx.ReportProgress(50, fmt.Sprintf("Awaiting inclusion: 0/%d exits included", len(pendingValidators)))

	for len(pendingValidators) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case block := <-blockSubscription.Channel():
			blockData := block.AwaitBlock(ctx, 2*time.Second)
			if blockData == nil {
				continue
			}

			exits, err := blockData.VoluntaryExits()
			if err != nil {
				t.logger.Warnf("could not get voluntary exits from block %v: %v", block.Slot, err)
				continue
			}

			for _, exit := range exits {
				if !pendingValidators[exit.Message.ValidatorIndex] {
					continue
				}

				delete(pendingValidators, exit.Message.ValidatorIndex)

				includedCount++

				t.ctx.Outputs.SetVar("includedExits", includedCount)
				t.logger.Infof("Voluntary exit for validator %d included in block %d (%d/%d)",
					exit.Message.ValidatorIndex, block.Slot, includedCount, totalCount)

				// Calculate progress: 50% for generation + 50% for inclusion
				inclusionProgress := float64(includedCount) / float64(totalCount) * 50
				t.ctx.ReportProgress(50+inclusionProgress,
					fmt.Sprintf("Awaiting inclusion: %d/%d exits included", includedCount, totalCount))
			}
		}
	}

	t.ctx.SetResult(types.TaskResultSuccess)
	t.ctx.ReportProgress(100, fmt.Sprintf("All %d voluntary exits included on-chain", totalCount))

	return nil
}

func (t *Task) loadChainState(ctx context.Context) (*phase0.Fork, error) {
	client := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().AwaitReadyEndpoint(ctx, consensus.AnyClient)
	if client == nil {
		return nil, ctx.Err()
	}

	fork, err := client.GetRPCClient().GetForkState(ctx, "head")
	if err != nil {
		return nil, err
	}

	return fork, nil
}

func (t *Task) generateVoluntaryExit(ctx context.Context, accountIdx uint64, fork *phase0.Fork) (phase0.ValidatorIndex, error) {
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, validatorKeyPath)
	if err != nil {
		return 0, fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	// select validator
	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()
	for _, val := range t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet() {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	// check validator status
	if validator == nil {
		return 0, fmt.Errorf("validator not found: 0x%x", validatorPubkey)
	}

	if validator.Validator.ExitEpoch != 18446744073709551615 {
		return 0, fmt.Errorf("validator %v is already exited", validator.Index)
	}

	// select client
	var client *consensus.Client

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetConsensusPool().GetReadyEndpoint(consensus.AnyClient)
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return 0, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		client = clients[0].ConsensusClient
	}

	// build voluntary exit message
	specs := clientPool.GetConsensusPool().GetBlockCache().GetSpecs()
	operation := &phase0.VoluntaryExit{
		ValidatorIndex: validator.Index,
	}

	if t.config.ExitEpoch >= 0 {
		operation.Epoch = phase0.Epoch(t.config.ExitEpoch) //nolint:gosec // no overflow possible
	} else {
		currentSlot, _ := client.GetLastHead()
		operation.Epoch = phase0.Epoch(currentSlot / phase0.Slot(specs.SlotsPerEpoch))
	}

	root, err := operation.HashTreeRoot()
	if err != nil {
		return 0, fmt.Errorf("failed to generate root for exit operation: %w", err)
	}

	var secKey hbls.SecretKey

	err = secKey.Deserialize(validatorPrivkey.Marshal())
	if err != nil {
		return 0, fmt.Errorf("failed converting validator priv key: %w", err)
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
		return 0, err
	}

	return validator.Index, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
