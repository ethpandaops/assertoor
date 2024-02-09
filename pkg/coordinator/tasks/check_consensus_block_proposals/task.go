package checkconsensusblockproposals

import (
	"context"
	"fmt"
	"math/big"
	"regexp"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
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
	ctx                 *types.TaskContext
	options             *types.TaskOptions
	config              Config
	logger              logrus.FieldLogger
	firstHeight         map[uint16]uint64
	currentValidatorSet map[uint64]*v1.Validator
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:         ctx,
		options:     options,
		logger:      ctx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
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
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool()

	blockSubscription := consensusPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	wallclockEpochSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer wallclockEpochSubscription.Unsubscribe()

	// load current epoch duties
	t.loadValidatorSet(ctx)

	totalMatches := 0

	for {
		select {
		case block := <-blockSubscription.Channel():
			matches := t.checkBlock(ctx, block)
			if matches {
				t.logger.Infof("matching block %v [0x%x]", block.Slot, block.Root)

				totalMatches++
			}

			if t.config.BlockCount > 0 {
				if totalMatches >= t.config.BlockCount {
					t.ctx.SetResult(types.TaskResultSuccess)
					return nil
				}
			} else {
				if matches {
					t.ctx.SetResult(types.TaskResultSuccess)
				} else {
					t.ctx.SetResult(types.TaskResultNone)
				}
			}
		case <-wallclockEpochSubscription.Channel():
			t.loadValidatorSet(ctx)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) loadValidatorSet(ctx context.Context) {
	client := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool().GetReadyEndpoint(consensus.UnspecifiedClient)
	validatorSet, err := client.GetRPCClient().GetStateValidators(ctx, "head")

	if err != nil {
		t.logger.Errorf("error while fetching validator set: %v", err.Error())
		return
	}

	t.currentValidatorSet = make(map[uint64]*v1.Validator)
	for _, val := range validatorSet {
		t.currentValidatorSet[uint64(val.Index)] = val
	}
}

//nolint:gocyclo // ignore
func (t *Task) checkBlock(ctx context.Context, block *consensus.Block) bool {
	blockData := block.AwaitBlock(ctx, 2*time.Second)
	if blockData == nil {
		t.logger.Warnf("could not fetch block data for block %v [0x%x]", block.Slot, block.Root)
		return false
	}

	// check graffiti
	if t.config.GraffitiPattern != "" && !t.checkBlockGraffiti(block, blockData) {
		return false
	}

	// check validator name
	if t.config.ValidatorNamePattern != "" && !t.checkBlockValidatorName(block, blockData) {
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

	validatorName := t.ctx.Scheduler.GetCoordinator().ValidatorNames().GetValidatorName(uint64(proposerIndex))

	matched, err := regexp.MatchString(t.config.ValidatorNamePattern, validatorName)
	if err != nil {
		t.logger.Warnf("could not check validator name for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if !matched {
		t.logger.Infof("check failed for block %v [0x%x]: unmatched validator name (have: %v, want: %v)", block.Slot, block.Root, validatorName, t.config.ValidatorNamePattern)
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
		if t.currentValidatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, pubkey := range t.config.ExpectExits {
			found := false

			for _, exit := range exits {
				validator := t.currentValidatorSet[uint64(exit.Message.ValidatorIndex)]
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
	attSlashings, err := blockData.AttesterSlashings()
	if err != nil {
		t.logger.Warnf("could not get attester slashings for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	propSlashings, err := blockData.ProposerSlashings()
	if err != nil {
		t.logger.Warnf("could not get attester slashings for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	slashingCount := len(attSlashings) + len(propSlashings)
	if slashingCount < t.config.MinSlashingCount {
		t.logger.Infof("check failed for block %v [0x%x]: not enough exits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinSlashingCount, slashingCount)
		return false
	}

	if len(t.config.ExpectSlashings) > 0 {
		if t.currentValidatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, expectedSlashing := range t.config.ExpectSlashings {
			found := false

			if !found && (expectedSlashing.SlashingType == "" || expectedSlashing.SlashingType == "attester") {
				for _, slashing := range attSlashings {
					inter := intersect.Simple(slashing.Attestation1.AttestingIndices, slashing.Attestation2.AttestingIndices)
					for _, j := range inter {
						valIdx, ok := j.(uint64)
						if !ok {
							continue
						}

						validator := t.currentValidatorSet[valIdx]
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
					valIdx := uint64(slashing.SignedHeader1.Message.ProposerIndex)

					validator := t.currentValidatorSet[valIdx]
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
				t.logger.Infof("check failed for block %v [0x%x]: expected deposit not found (pubkey: %v)", block.Slot, block.Root, expectedSlashing.PublicKey)
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
		t.logger.Infof("check failed for block %v [0x%x]: not enough exits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinAttesterSlashingCount, slashingCount)
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
		t.logger.Infof("check failed for block %v [0x%x]: not enough exits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinProposerSlashingCount, slashingCount)
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
		if t.currentValidatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, expectedBlsChange := range t.config.ExpectBlsChanges {
			found := false

			for _, blsChange := range blsChanges {
				validator := t.currentValidatorSet[uint64(blsChange.Message.ValidatorIndex)]
				if validator == nil {
					continue
				}

				if validator.Validator.PublicKey.String() == expectedBlsChange.PublicKey {
					if expectedBlsChange.Address != "" && expectedBlsChange.Address != blsChange.Message.ToExecutionAddress.String() {
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
		if t.currentValidatorSet == nil {
			t.logger.Errorf("check failed: no validator set")
			return false
		}

		for _, expectedWithdrawal := range t.config.ExpectWithdrawals {
			found := false

			for _, withdrawal := range withdrawals {
				validator := t.currentValidatorSet[uint64(withdrawal.ValidatorIndex)]
				if validator == nil {
					continue
				}

				if validator.Validator.PublicKey.String() == expectedWithdrawal.PublicKey {
					withdrawalAmount := big.NewInt(int64(withdrawal.Amount))
					withdrawalAmount = withdrawalAmount.Mul(withdrawalAmount, big.NewInt(1000000000))

					switch {
					case expectedWithdrawal.Address != "" && expectedWithdrawal.Address != withdrawal.Address.String():
						t.logger.Warnf("check failed: withdrawal found, but execution address does not match (have: %v, want: %v)", withdrawal.Address.String(), expectedWithdrawal.Address)
					case expectedWithdrawal.MinAmount.Cmp(big.NewInt(0)) > 0 && expectedWithdrawal.MinAmount.Cmp(withdrawalAmount) < 0:
						t.logger.Warnf("check failed: withdrawal found, but amount lower than minimum (have: %v, want >= %v)", withdrawalAmount, expectedWithdrawal.MinAmount)
					default:
						found = true
					}

					break
				}
			}

			if !found {
				t.logger.Infof("check failed for block %v [0x%x]: expected bls change not found (pubkey: %v)", block.Slot, block.Root, expectedWithdrawal.PublicKey)
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
