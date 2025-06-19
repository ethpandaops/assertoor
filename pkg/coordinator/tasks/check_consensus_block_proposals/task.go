package checkconsensusblockproposals

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/erigontech/assertoor/pkg/coordinator/clients/consensus"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	"github.com/ethereum/go-ethereum/common"
	"github.com/juliangruber/go-intersect"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_block_proposals"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check for consensus block proposals that meet specific criteria.",
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

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	blockSubscription := consensusPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	totalMatches := 0
	matchingBlocks := []*consensus.Block{}

	checkBlockMatch := func(block *consensus.Block) bool {
		matches := t.checkBlock(ctx, block)
		if matches {
			matchingBlocks = append(matchingBlocks, block)
			t.logger.Infof("matching block %v [0x%x]", block.Slot, block.Root)

			totalMatches++
		}

		if t.config.BlockCount > 0 {
			if totalMatches >= t.config.BlockCount {
				t.setMatchingBlocksOutput(matchingBlocks)
				t.ctx.SetResult(types.TaskResultSuccess)

				return true
			}
		} else {
			if matches {
				t.ctx.SetResult(types.TaskResultSuccess)
			} else {
				t.ctx.SetResult(types.TaskResultNone)
			}
		}

		return false
	}

	// check current block
	if t.config.CheckLookback > 0 {
		if blocks := consensusPool.GetBlockCache().GetCachedBlocks(); len(blocks) > 0 {
			lookbackBlocks := []*consensus.Block{}
			block := blocks[0]

			for {
				lookbackBlocks = append(lookbackBlocks, block)
				if len(lookbackBlocks) >= t.config.CheckLookback {
					break
				}

				parentRoot := block.GetParentRoot()
				if parentRoot == nil {
					break
				}

				block = consensusPool.GetBlockCache().GetCachedBlockByRoot(*parentRoot)
				if block == nil {
					break
				}
			}

			for i := len(lookbackBlocks) - 1; i >= 0; i-- {
				if checkBlockMatch(lookbackBlocks[i]) {
					return nil
				}
			}
		}
	}

	for {
		select {
		case block := <-blockSubscription.Channel():
			if checkBlockMatch(block) {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) setMatchingBlocksOutput(blocks []*consensus.Block) {
	blockRoots := []string{}
	blockHeaders := []any{}
	blockBodies := []any{}

	for _, block := range blocks {
		blockRoots = append(blockRoots, block.Root.String())

		var blockHeader, blockBody any

		if header, err := vars.GeneralizeData(block.GetHeader()); err == nil {
			blockHeader = header
		} else {
			t.logger.Warnf("Failed encoding block #%v header for matchingBlockHeaders output: %v", block.Slot, err)
		}

		if body, err := vars.GeneralizeData(consensus.GetBlockBody(block.GetBlock())); err == nil {
			blockBody = body
		} else {
			t.logger.Warnf("Failed encoding block #%v header for matchingBlockHeaders output: %v", block.Slot, err)
		}

		blockHeaders = append(blockHeaders, blockHeader)
		blockBodies = append(blockBodies, blockBody)
	}

	t.ctx.Outputs.SetVar("matchingBlockRoots", blockRoots)
	t.ctx.Outputs.SetVar("matchingBlockHeaders", blockHeaders)
	t.ctx.Outputs.SetVar("matchingBlockBodies", blockBodies)
}

//nolint:gocyclo // ignore
func (t *Task) checkBlock(ctx context.Context, block *consensus.Block) bool {
	blockData := block.AwaitBlock(ctx, 2*time.Second)
	if blockData == nil {
		t.logger.Warnf("could not fetch block data for block %v [0x%x]", block.Slot, block.Root)
		return false
	}

	// check validator name
	if t.config.ValidatorNamePattern != "" && !t.checkBlockValidatorName(block, blockData) {
		return false
	}

	// check graffiti
	if t.config.GraffitiPattern != "" && !t.checkBlockGraffiti(block, blockData) {
		return false
	}

	// check extra data
	if t.config.ExtraDataPattern != "" && !t.checkBlockExtraData(block, blockData) {
		return false
	}

	// check attestation count
	if t.config.MinAttestationCount > 0 && !t.checkBlockAttestations(block, blockData) {
		return false
	}

	// check deposit count
	if (t.config.MinDepositCount > 0 || len(t.config.ExpectDeposits) > 0) && !t.checkBlockDeposits(block, blockData) {
		return false
	}

	// check exit count
	if (t.config.MinExitCount > 0 || len(t.config.ExpectExits) > 0) && !t.checkBlockExits(block, blockData) {
		return false
	}

	// check slashing count
	if (t.config.MinSlashingCount > 0 || len(t.config.ExpectSlashings) > 0) && !t.checkBlockSlashings(block, blockData) {
		return false
	}

	// check attester slashing count
	if t.config.MinAttesterSlashingCount > 0 && !t.checkBlockAttesterSlashings(block, blockData) {
		return false
	}

	// check proposer slashing count
	if t.config.MinProposerSlashingCount > 0 && !t.checkBlockProposerSlashings(block, blockData) {
		return false
	}

	// check bls change count
	if (t.config.MinBlsChangeCount > 0 || len(t.config.ExpectBlsChanges) > 0) && !t.checkBlockBlsChanges(block, blockData) {
		return false
	}

	// check withdrawal count
	if (t.config.MinWithdrawalCount > 0 || len(t.config.ExpectWithdrawals) > 0) && !t.checkBlockWithdrawals(block, blockData) {
		return false
	}

	// check transaction count
	if t.config.MinTransactionCount > 0 && !t.checkBlockTransactions(block, blockData) {
		return false
	}

	// check blob count
	if t.config.MinBlobCount > 0 && !t.checkBlockBlobs(block, blockData) {
		return false
	}

	// check deposit request count
	if (t.config.MinDepositRequestCount > 0 || len(t.config.ExpectDepositRequests) > 0) && !t.checkBlockDepositRequests(block, blockData) {
		return false
	}

	// check withdrawal request count
	if (t.config.MinWithdrawalRequestCount > 0 || len(t.config.ExpectWithdrawalRequests) > 0) && !t.checkBlockWithdrawalRequests(block, blockData) {
		return false
	}

	// check consolidation request count
	if (t.config.MinConsolidationRequestCount > 0 || len(t.config.ExpectConsolidationRequests) > 0) && !t.checkBlockConsolidationRequests(block, blockData) {
		return false
	}

	return true
}

func (t *Task) checkBlockGraffiti(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	graffiti, err := blockData.Graffiti()
	if err != nil {
		t.logger.Warnf("could not get graffiti for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	matched, err := regexp.MatchString(t.config.GraffitiPattern, string(graffiti[:]))
	if err != nil {
		t.logger.Warnf("could not check graffiti for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if !matched {
		t.logger.Infof("check failed for block %v [0x%x]: unmatched graffiti", block.Slot, block.Root)
		return false
	}

	return true
}

func (t *Task) checkBlockValidatorName(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	proposerIndex, err := blockData.ProposerIndex()
	if err != nil {
		t.logger.Warnf("could not get proposer index for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	validatorName := t.ctx.Scheduler.GetServices().ValidatorNames().GetValidatorName(uint64(proposerIndex))

	matched, err := regexp.MatchString(t.config.ValidatorNamePattern, validatorName)
	if err != nil {
		t.logger.Warnf("could not check validator name for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if !matched {
		t.logger.Debugf("check failed for block %v [0x%x]: unmatched validator name (have: %v, want: %v)", block.Slot, block.Root, validatorName, t.config.ValidatorNamePattern)
		return false
	}

	return true
}

func (t *Task) checkBlockExtraData(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	extraData, err := consensus.GetExecutionExtraData(blockData)
	if err != nil {
		t.logger.Warnf("could not get extra data for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	matched, err := regexp.MatchString(t.config.ExtraDataPattern, string(extraData))
	if err != nil {
		t.logger.Warnf("could not check extra data for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if !matched {
		t.logger.Infof("check failed for block %v [0x%x]: unmatched extra data", block.Slot, block.Root)
		return false
	}

	return true
}

func (t *Task) checkBlockAttestations(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	attestations, err := blockData.Attestations()
	if err != nil {
		t.logger.Warnf("could not get attestations for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(attestations) < t.config.MinAttestationCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough attestations (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinAttestationCount, len(attestations))
		return false
	}

	return true
}

func (t *Task) checkBlockDeposits(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	deposits, err := blockData.Deposits()
	if err != nil {
		t.logger.Warnf("could not get deposits for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(deposits) < t.config.MinDepositCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough deposits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinDepositCount, len(deposits))
		return false
	}

	if len(t.config.ExpectDeposits) > 0 {
		for _, pubkey := range t.config.ExpectDeposits {
			found := false

			for _, deposit := range deposits {
				if deposit.Data.PublicKey.String() == pubkey {
					found = true
					break
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected deposit not found (pubkey: %v)", block.Slot, block.Root, pubkey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockExits(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	exits, err := blockData.VoluntaryExits()
	if err != nil {
		t.logger.Warnf("could not get voluntary exits for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(exits) < t.config.MinExitCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough exits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinExitCount, len(exits))
		return false
	}

	if len(t.config.ExpectExits) > 0 {
		validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()
		if validatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, pubkey := range t.config.ExpectExits {
			found := false

			for _, exit := range exits {
				validator := validatorSet[exit.Message.ValidatorIndex]
				if validator == nil {
					continue
				}

				if validator.Validator.PublicKey.String() == pubkey {
					found = true
					break
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected exit not found (pubkey: %v)", block.Slot, block.Root, pubkey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockSlashings(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	attSlashingsVersioned, err := blockData.AttesterSlashings()
	if err != nil {
		t.logger.Warnf("could not get attester slashings for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	propSlashings, err := blockData.ProposerSlashings()
	if err != nil {
		t.logger.Warnf("could not get attester slashings for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	slashingCount := len(attSlashingsVersioned) + len(propSlashings)
	if slashingCount < t.config.MinSlashingCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough slashings (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinSlashingCount, slashingCount)
		return false
	}

	if len(t.config.ExpectSlashings) > 0 {
		validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()
		if validatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, expectedSlashing := range t.config.ExpectSlashings {
			found := false

			if !found && (expectedSlashing.SlashingType == "" || expectedSlashing.SlashingType == "attester") {
				for _, slashing := range attSlashingsVersioned {
					att1, err1 := slashing.Attestation1()
					att2, err2 := slashing.Attestation2()

					if err1 != nil || err2 != nil {
						continue
					}

					att1indices, err1 := att1.AttestingIndices()
					att2indices, err2 := att2.AttestingIndices()

					if err1 != nil || err2 != nil {
						continue
					}

					inter := intersect.Simple(att1indices, att2indices)
					for _, j := range inter {
						valIdx, ok := j.(uint64)
						if !ok {
							continue
						}

						validator := validatorSet[phase0.ValidatorIndex(valIdx)]
						if validator == nil {
							continue
						}

						if validator.Validator.PublicKey.String() == expectedSlashing.PublicKey {
							found = true
							break
						}
					}

					if found {
						break
					}
				}
			}

			if !found && (expectedSlashing.SlashingType == "" || expectedSlashing.SlashingType == "proposer") {
				for _, slashing := range propSlashings {
					validator := validatorSet[slashing.SignedHeader1.Message.ProposerIndex]
					if validator == nil {
						continue
					}

					if validator.Validator.PublicKey.String() == expectedSlashing.PublicKey {
						found = true
						break
					}
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected slashing not found (pubkey: %v)", block.Slot, block.Root, expectedSlashing.PublicKey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockAttesterSlashings(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	attSlashings, err := blockData.AttesterSlashings()
	if err != nil {
		t.logger.Warnf("could not get attester slashings for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	slashingCount := len(attSlashings)
	if slashingCount < t.config.MinAttesterSlashingCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough attester slashings (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinAttesterSlashingCount, slashingCount)
		return false
	}

	return true
}

func (t *Task) checkBlockProposerSlashings(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	propSlashings, err := blockData.ProposerSlashings()
	if err != nil {
		t.logger.Warnf("could not get attester slashings for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	slashingCount := len(propSlashings)
	if slashingCount < t.config.MinProposerSlashingCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough proposer slashings (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinProposerSlashingCount, slashingCount)
		return false
	}

	return true
}

func (t *Task) checkBlockBlsChanges(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	blsChanges, err := blockData.BLSToExecutionChanges()
	if err != nil {
		t.logger.Warnf("could not get bls to execution changes for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(blsChanges) < t.config.MinBlsChangeCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough bls changes (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinBlsChangeCount, len(blsChanges))
		return false
	}

	if len(t.config.ExpectBlsChanges) > 0 {
		validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()
		if validatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, expectedBlsChange := range t.config.ExpectBlsChanges {
			found := false

			for _, blsChange := range blsChanges {
				validator := validatorSet[blsChange.Message.ValidatorIndex]
				if validator == nil {
					continue
				}

				if validator.Validator.PublicKey.String() == expectedBlsChange.PublicKey {
					if expectedBlsChange.Address != "" && !strings.EqualFold(expectedBlsChange.Address, blsChange.Message.ToExecutionAddress.String()) {
						t.logger.Warnf("check failed: bls change found, but execution address does not match (have: %v, want: %v)", blsChange.Message.ToExecutionAddress.String(), expectedBlsChange.Address)
					} else {
						found = true
					}

					break
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected bls change not found (pubkey: %v)", block.Slot, block.Root, expectedBlsChange.PublicKey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockWithdrawals(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	withdrawals, err := blockData.Withdrawals()
	if err != nil {
		t.logger.Warnf("could not get withdrawals for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(withdrawals) < t.config.MinWithdrawalCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough withdrawals (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinWithdrawalCount, len(withdrawals))
		return false
	}

	if len(t.config.ExpectWithdrawals) > 0 {
		validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()
		if validatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, expectedWithdrawal := range t.config.ExpectWithdrawals {
			found := false

			for _, withdrawal := range withdrawals {
				validator := validatorSet[withdrawal.ValidatorIndex]
				if validator == nil {
					continue
				}

				if validator.Validator.PublicKey.String() == expectedWithdrawal.PublicKey {
					withdrawalAmount := big.NewInt(0).SetUint64(uint64(withdrawal.Amount))

					switch {
					case expectedWithdrawal.Address != "" && !strings.EqualFold(expectedWithdrawal.Address, withdrawal.Address.String()):
						t.logger.Warnf("check failed: withdrawal found, but execution address does not match (have: %v, want: %v)", withdrawal.Address.String(), expectedWithdrawal.Address)
					case expectedWithdrawal.MinAmount != nil && expectedWithdrawal.MinAmount.Cmp(big.NewInt(0)) > 0 && expectedWithdrawal.MinAmount.Cmp(withdrawalAmount) > 0:
						t.logger.Warnf("check failed: withdrawal found, but amount lower than minimum (have: %v, want >= %v)", withdrawalAmount, expectedWithdrawal.MinAmount)
					case expectedWithdrawal.MaxAmount != nil && expectedWithdrawal.MaxAmount.Cmp(big.NewInt(0)) > 0 && expectedWithdrawal.MaxAmount.Cmp(withdrawalAmount) < 0:
						t.logger.Warnf("check failed: withdrawal found, but amount higher than maximum (have: %v, want <= %v)", withdrawalAmount, expectedWithdrawal.MaxAmount)
					default:
						found = true
					}

					if found {
						break
					}
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected withdrawal not found (pubkey: %v)", block.Slot, block.Root, expectedWithdrawal.PublicKey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockTransactions(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	transactions, err := blockData.ExecutionTransactions()
	if err != nil {
		t.logger.Warnf("could not get transactions for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(transactions) < t.config.MinTransactionCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough transactions (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinTransactionCount, len(transactions))
		return false
	}

	return true
}

func (t *Task) checkBlockBlobs(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	blobs, err := blockData.BlobKZGCommitments()
	if err != nil {
		t.logger.Warnf("could not get blobs for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if len(blobs) < t.config.MinBlobCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough blobs (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinBlobCount, len(blobs))
		return false
	}

	return true
}

func (t *Task) checkBlockDepositRequests(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	executionRequests, err := blockData.ExecutionRequests()
	if err != nil {
		t.logger.Warnf("could not get execution requests for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	depositRequests := executionRequests.Deposits
	if len(depositRequests) < t.config.MinDepositRequestCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough deposit requests (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinDepositRequestCount, len(depositRequests))
		return false
	}

	if len(t.config.ExpectDepositRequests) > 0 {
		for _, expectedDepositRequest := range t.config.ExpectDepositRequests {
			found := false

			var expectedWithdrawalCreds []byte

			if expectedDepositRequest.WithdrawalCredentials != "" {
				expectedWithdrawalCreds = common.FromHex(expectedDepositRequest.WithdrawalCredentials)
			}

		requestLoop:
			for _, depositRequest := range depositRequests {
				if expectedDepositRequest.PublicKey == "" || depositRequest.Pubkey.String() == expectedDepositRequest.PublicKey {
					depositAmount := big.NewInt(0).SetUint64(uint64(depositRequest.Amount))

					switch {
					case expectedDepositRequest.WithdrawalCredentials != "" && !bytes.Equal(expectedWithdrawalCreds, depositRequest.WithdrawalCredentials):
						t.logger.Warnf("check failed: deposit request found, but withdrawal credentials do not match (have: 0x%x, want: 0x%x)", depositRequest.WithdrawalCredentials, expectedWithdrawalCreds)
					case expectedDepositRequest.Amount.Cmp(big.NewInt(0)) > 0 && expectedDepositRequest.Amount.Cmp(depositAmount) != 0:
						t.logger.Warnf("check failed: deposit request found, but amount does not match (have: %v, want: %v)", depositAmount, expectedDepositRequest.Amount.String())
					default:
						found = true
						break requestLoop
					}
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected deposit request not found (pubkey: %v)", block.Slot, block.Root, expectedDepositRequest.PublicKey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockWithdrawalRequests(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	executionRequests, err := blockData.ExecutionRequests()
	if err != nil {
		t.logger.Warnf("could not get execution requests for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	withdrawalRequests := executionRequests.Withdrawals
	if len(withdrawalRequests) < t.config.MinWithdrawalRequestCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough withdrawal requests (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinWithdrawalRequestCount, len(withdrawalRequests))
		return false
	}

	if len(t.config.ExpectWithdrawalRequests) > 0 {
		for _, expectedWithdrawalRequest := range t.config.ExpectWithdrawalRequests {
			found := false

			var expectedAddress, expectedPubKey []byte

			if expectedWithdrawalRequest.SourceAddress != "" {
				expectedAddress = common.FromHex(expectedWithdrawalRequest.SourceAddress)
			}

			if expectedWithdrawalRequest.ValidatorPubkey != "" {
				expectedPubKey = common.FromHex(expectedWithdrawalRequest.ValidatorPubkey)
			}

		requestLoop:
			for _, withdrawalRequest := range withdrawalRequests {
				if expectedWithdrawalRequest.ValidatorPubkey == "" || bytes.Equal(withdrawalRequest.ValidatorPubkey[:], expectedPubKey) {
					withdrawalAmount := big.NewInt(0).SetUint64(uint64(withdrawalRequest.Amount))

					switch {
					case expectedWithdrawalRequest.SourceAddress != "" && !bytes.Equal(expectedAddress, withdrawalRequest.SourceAddress[:]):
						t.logger.Warnf("check failed: withdrawal request found, but source address does not match (have: 0x%x, want: 0x%x)", withdrawalRequest.SourceAddress, expectedAddress)
					case expectedWithdrawalRequest.Amount != nil && expectedWithdrawalRequest.Amount.Cmp(withdrawalAmount) != 0:
						t.logger.Warnf("check failed: deposit request found, but amount does not match (have: %v, want: %v)", withdrawalAmount, expectedWithdrawalRequest.Amount.String())
					default:
						found = true
						break requestLoop
					}
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected withdrawal request not found (address: %v, pubkey: %v)", block.Slot, block.Root, expectedWithdrawalRequest.SourceAddress, expectedWithdrawalRequest.ValidatorPubkey)
				return false
			}
		}
	}

	return true
}

func (t *Task) checkBlockConsolidationRequests(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	executionRequests, err := blockData.ExecutionRequests()
	if err != nil {
		t.logger.Warnf("could not get execution requests for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	consolidationRequests := executionRequests.Consolidations
	if len(consolidationRequests) < t.config.MinConsolidationRequestCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough consolidation requests (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinConsolidationRequestCount, len(consolidationRequests))
		return false
	}

	if len(t.config.ExpectConsolidationRequests) > 0 {
		for _, expectedConsolidationRequest := range t.config.ExpectConsolidationRequests {
			found := false

			var expectedAddress, expectedSrcPubKey, expectedTgtPubKey []byte

			if expectedConsolidationRequest.SourceAddress != "" {
				expectedAddress = common.FromHex(expectedConsolidationRequest.SourceAddress)
			}

			if expectedConsolidationRequest.SourcePubkey != "" {
				expectedSrcPubKey = common.FromHex(expectedConsolidationRequest.SourcePubkey)
			}

			if expectedConsolidationRequest.TargetPubkey != "" {
				expectedTgtPubKey = common.FromHex(expectedConsolidationRequest.TargetPubkey)
			}

		requestLoop:
			for _, consolidationRequest := range consolidationRequests {
				if expectedConsolidationRequest.SourcePubkey == "" || bytes.Equal(consolidationRequest.SourcePubkey[:], expectedSrcPubKey) {
					switch {
					case expectedConsolidationRequest.SourceAddress != "" && !bytes.Equal(expectedAddress, consolidationRequest.SourceAddress[:]):
						t.logger.Warnf("check failed: consolidation request found, but source address does not match (have: 0x%x, want: 0x%x)", consolidationRequest.SourceAddress, expectedAddress)
					case expectedConsolidationRequest.TargetPubkey != "" && !bytes.Equal(expectedTgtPubKey, consolidationRequest.TargetPubkey[:]):
						t.logger.Warnf("check failed: consolidation request found, but target pubkey does not match (have: 0x%x, want: 0x%x)", consolidationRequest.SourceAddress, expectedAddress)

					default:
						found = true
						break requestLoop
					}
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected consolidation request not found (address: %v, pubkey: %v)", block.Slot, block.Root, expectedConsolidationRequest.SourceAddress, expectedConsolidationRequest.SourcePubkey)
				return false
			}
		}
	}

	return true
}
