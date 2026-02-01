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
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
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
		Category:    "validator",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "blsChanges",
				Type:        "array",
				Description: "Array of generated BLS change operations.",
			},
			{
				Name:        "latestBlsChange",
				Type:        "object",
				Description: "The most recently generated BLS change operation.",
			},
			{
				Name:        "includedBlsChanges",
				Type:        "number",
				Description: "Number of BLS changes included on-chain (when awaitInclusion is enabled).",
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
	targetAddr common.Eth1Address
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

	err = t.targetAddr.UnmarshalText([]byte(config.TargetAddress))
	if err != nil {
		return fmt.Errorf("cannot decode execution addr: %w", err)
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

	perSlotCount := 0
	totalCount := 0
	blsChangesList := []any{}

	// Track submitted validator indices for awaitInclusion
	pendingValidators := make(map[phase0.ValidatorIndex]bool)

	// Calculate target count for progress reporting
	targetCount := 0
	if t.config.LimitTotal > 0 {
		targetCount = t.config.LimitTotal
	} else if t.lastIndex > 0 {
		targetCount = int(t.lastIndex - t.nextIndex) //nolint:gosec // no overflow possible
	}

	t.ctx.ReportProgress(0, "Starting BLS change generation")

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		blsChange, validatorIndex, err := t.generateBlsChange(ctx, accountIdx)
		if err != nil {
			t.logger.Errorf("error generating bls change: %v", err.Error())
		} else {
			blsChangesList = append(blsChangesList, blsChange)
			t.ctx.Outputs.SetVar("blsChanges", blsChangesList)

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
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d BLS changes", totalCount, targetCount))
			} else {
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d BLS changes", totalCount))
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

	t.ctx.Outputs.SetVar("blsChanges", blsChangesList)

	// Await inclusion in blocks if configured
	if t.config.AwaitInclusion && len(pendingValidators) > 0 {
		err := t.awaitInclusion(ctx, pendingValidators, totalCount)
		if err != nil {
			return err
		}
	} else {
		t.ctx.ReportProgress(100, fmt.Sprintf("Completed generating %d BLS changes", totalCount))
	}

	return nil
}

func (t *Task) awaitInclusion(ctx context.Context, pendingValidators map[phase0.ValidatorIndex]bool, totalCount int) error {
	blockSubscription := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	includedCount := 0
	t.ctx.Outputs.SetVar("includedBlsChanges", includedCount)

	t.logger.Infof("waiting for %d BLS changes to be included in blocks", len(pendingValidators))
	t.ctx.ReportProgress(50, fmt.Sprintf("Awaiting inclusion: 0/%d BLS changes included", len(pendingValidators)))

	for len(pendingValidators) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case block := <-blockSubscription.Channel():
			blockData := block.AwaitBlock(ctx, 2*time.Second)
			if blockData == nil {
				continue
			}

			blsChanges, err := blockData.BLSToExecutionChanges()
			if err != nil {
				t.logger.Warnf("could not get BLS changes from block %v: %v", block.Slot, err)
				continue
			}

			for _, blsChange := range blsChanges {
				if !pendingValidators[blsChange.Message.ValidatorIndex] {
					continue
				}

				delete(pendingValidators, blsChange.Message.ValidatorIndex)

				includedCount++

				t.ctx.Outputs.SetVar("includedBlsChanges", includedCount)
				t.logger.Infof("BLS change for validator %d included in block %d (%d/%d)",
					blsChange.Message.ValidatorIndex, block.Slot, includedCount, totalCount)

				// Calculate progress: 50% for generation + 50% for inclusion
				inclusionProgress := float64(includedCount) / float64(totalCount) * 50
				t.ctx.ReportProgress(50+inclusionProgress,
					fmt.Sprintf("Awaiting inclusion: %d/%d BLS changes included", includedCount, totalCount))
			}
		}
	}

	t.ctx.SetResult(types.TaskResultSuccess)
	t.ctx.ReportProgress(100, fmt.Sprintf("All %d BLS changes included on-chain", totalCount))

	return nil
}

func (t *Task) generateBlsChange(ctx context.Context, accountIdx uint64) (any, phase0.ValidatorIndex, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, validatorKeyPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()

	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()

	t.logger.Debugf("generated validator pubkey %v: 0x%x", validatorKeyPath, validatorPubkey)

	for _, val := range validatorSet {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if validator == nil {
		return nil, 0, fmt.Errorf("validator not found")
	}

	if validator.Validator.WithdrawalCredentials[0] != 0x00 {
		return nil, 0, fmt.Errorf("validator %v does not have 0x00 withdrawal creds", validator.Index)
	}

	withdrAccPath := fmt.Sprintf("m/12381/3600/%d/0", accountIdx)

	withdr, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, withdrAccPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed generating key %v: %w", withdrAccPath, err)
	}

	var withdrPub common.BLSPubkey

	copy(withdrPub[:], withdr.PublicKey().Marshal())

	t.logger.Debugf("generated withdrawal pubkey %v: 0x%x", withdrAccPath, withdrPub[:])

	msg := common.BLSToExecutionChange{
		ValidatorIndex:     common.ValidatorIndex(validator.Index),
		FromBLSPubKey:      withdrPub,
		ToExecutionAddress: t.targetAddr,
	}

	var secKey hbls.SecretKey

	err = secKey.Deserialize(withdr.Marshal())
	if err != nil {
		return nil, 0, fmt.Errorf("failed converting validator priv key: %w", err)
	}

	msgRoot := msg.HashTreeRoot(tree.GetHashFn())
	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(common.DOMAIN_BLS_TO_EXECUTION_CHANGE, common.Version(genesis.GenesisForkVersion), tree.Root(genesis.GenesisValidatorsRoot))
	signingRoot := common.ComputeSigningRoot(msgRoot, dom)
	sig := secKey.SignHash(signingRoot[:])

	var signedMsg common.SignedBLSToExecutionChange

	signedMsg.BLSToExecutionChange = msg
	copy(signedMsg.Signature[:], sig.Serialize())

	signedChangeJSON, err := json.Marshal(&signedMsg)
	if err != nil {
		return nil, 0, err
	}

	var blsChangeRes any
	if blsc, err2 := vars.GeneralizeData(signedMsg); err2 == nil {
		blsChangeRes = blsc
	} else {
		t.logger.Warnf("Failed encoding blschange output: %v", err2)
	}

	t.ctx.Outputs.SetVar("latestBlsChange", blsChangeRes)

	var client *consensus.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetConsensusPool().AwaitReadyEndpoint(ctx, consensus.AnyClient)
		if client == nil {
			return nil, 0, ctx.Err()
		}
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return nil, 0, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		client = clients[0].ConsensusClient
	}

	blsChange := &capella.SignedBLSToExecutionChange{}

	err = json.Unmarshal(signedChangeJSON, blsChange)
	if err != nil {
		return nil, 0, err
	}

	t.logger.WithField("client", client.GetName()).Infof("sending bls change for validator %v", validator.Index)

	err = client.GetRPCClient().SubmitBLSToExecutionChanges(ctx, []*capella.SignedBLSToExecutionChange{blsChange})
	if err != nil {
		return nil, 0, err
	}

	return blsChangeRes, validator.Index, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
