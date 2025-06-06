package checkconsensusattestationstats

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_attestation_stats"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check attestation stats for consensus chain.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx             *types.TaskContext
	options         *types.TaskOptions
	config          Config
	logger          logrus.FieldLogger
	attesterDutyMap map[uint64]map[phase0.Root]*attesterDuties
	passedEpochs    uint64
}

type attesterDuties struct {
	validatorCount   uint64
	validatorBalance uint64
	duties           map[string][]*attesterDuty
}

type attesterDuty struct {
	validator uint64
	balance   uint64
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

	wallclockSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer wallclockSubscription.Unsubscribe()

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return fmt.Errorf("failed fetching wallclock: %w", err)
	}

	// start vote counting from next epoch as current epoch might be incomplete
	lastCheckedEpoch := currentEpoch.Number()

	t.logger.Infof("current epoch: %v, starting attestation aggregation at epoch %v", lastCheckedEpoch, lastCheckedEpoch+1)

	t.attesterDutyMap = map[uint64]map[phase0.Root]*attesterDuties{}
	defer func() {
		t.attesterDutyMap = nil
	}()

	// set cache follow distance to at least the last 4 epochs, so we can safely aggregate voting stats for epoch n-2
	specs := consensusPool.GetBlockCache().GetSpecs()
	consensusPool.GetBlockCache().SetMinFollowDistance(specs.SlotsPerEpoch * 4)

	for {
		select {
		case block := <-blockSubscription.Channel():
			t.processBlock(ctx, block)

		case currentEpoch := <-wallclockSubscription.Channel():
			epoch := currentEpoch.Number()

			checkEpoch := epoch - 2
			if epoch < 2 || checkEpoch <= lastCheckedEpoch {
				break
			}

			t.runAttestationStatsCheck(ctx, checkEpoch)

			lastCheckedEpoch = checkEpoch

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) processBlock(ctx context.Context, block *consensus.Block) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	specs := consensusPool.GetBlockCache().GetSpecs()

	blockBody := block.AwaitBlock(ctx, 500*time.Millisecond)
	if blockBody == nil {
		return
	}

	parentBlock := consensusPool.GetBlockCache().GetCachedBlockByRoot(*block.GetParentRoot())
	if parentBlock == nil {
		return
	}

	currentBlockEpoch := uint64(block.Slot) / specs.SlotsPerEpoch
	parentBlockEpoch := uint64(parentBlock.Slot) / specs.SlotsPerEpoch

	if parentBlockEpoch == currentBlockEpoch {
		return
	}

	parentStateRoot := parentBlock.GetHeader().Message.StateRoot

	if t.attesterDutyMap[currentBlockEpoch] == nil {
		t.attesterDutyMap[currentBlockEpoch] = map[phase0.Root]*attesterDuties{}
	} else if t.attesterDutyMap[currentBlockEpoch][parentBlock.Root] != nil {
		return
	}

	t.logger.Infof("loading epoch %v duties (dependent: %v)", currentBlockEpoch, parentBlock.Root.String())

	validators := consensusPool.GetValidatorSet()
	if validators == nil {
		t.logger.Warnf("could not load validators", currentBlockEpoch, parentStateRoot.String())
		return
	}

	committees, err := block.GetSeenBy()[0].GetRPCClient().GetCommitteeDuties(ctx, parentStateRoot.String(), currentBlockEpoch)
	if err != nil {
		t.logger.Warnf("could not load epoch %v committees", currentBlockEpoch, parentStateRoot.String())
		return
	}

	attesterDuties := &attesterDuties{
		duties: map[string][]*attesterDuty{},
	}

	for _, validator := range validators {
		if uint64(validator.Validator.ActivationEpoch) <= currentBlockEpoch && currentBlockEpoch < uint64(validator.Validator.ExitEpoch) {
			attesterDuties.validatorCount++
			attesterDuties.validatorBalance += uint64(validator.Validator.EffectiveBalance)
		}
	}

	for _, committee := range committees {
		for _, valIndex := range committee.Validators {
			valIndexU64 := uint64(valIndex)

			k := fmt.Sprintf("%v-%v", uint64(committee.Slot), uint64(committee.Index))
			if attesterDuties.duties[k] == nil {
				attesterDuties.duties[k] = make([]*attesterDuty, 0)
			}

			validator := validators[valIndex]
			attesterDuties.duties[k] = append(attesterDuties.duties[k], &attesterDuty{
				validator: valIndexU64,
				balance:   uint64(validator.Validator.EffectiveBalance),
			})
		}
	}

	t.attesterDutyMap[currentBlockEpoch][parentBlock.Root] = attesterDuties
}

func (t *Task) runAttestationStatsCheck(ctx context.Context, epoch uint64) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	canonicalFork := consensusPool.GetCanonicalFork(1)

	epochVotes := t.aggregateEpochVotes(ctx, epoch)
	delete(t.attesterDutyMap, epoch)

	for _, epochVote := range epochVotes {
		isCanonical := consensusPool.GetBlockCache().IsCanonicalBlock(epochVote.headRoot, canonicalFork.Root)
		if !isCanonical {
			continue
		}

		result := t.checkEpochVotes(epoch, epochVote)

		if result {
			t.passedEpochs++
			if t.passedEpochs >= t.config.MinCheckedEpochs {
				t.ctx.SetResult(types.TaskResultSuccess)
			}
		} else {
			t.passedEpochs = 0
			if t.config.FailOnCheckMiss {
				t.ctx.SetResult(types.TaskResultFailure)
			} else {
				t.ctx.SetResult(types.TaskResultNone)
			}
		}

		t.logger.Infof("epoch %v attestation check result: %v. passed checks: %v, want: %v", epoch, result, t.passedEpochs, t.config.MinCheckedEpochs)

		break
	}
}

func (t *Task) checkEpochVotes(epoch uint64, epochVote *epochVotes) bool {
	targetPercent := float64(epochVote.currentEpoch.targetVoteAmount+epochVote.nextEpoch.targetVoteAmount) * 100.0 / float64(epochVote.attesterDuties.validatorBalance)
	headPercent := float64(epochVote.currentEpoch.headVoteAmount+epochVote.nextEpoch.headVoteAmount) * 100.0 / float64(epochVote.attesterDuties.validatorBalance)
	totalPercent := float64(epochVote.currentEpoch.totalVoteAmount+epochVote.nextEpoch.totalVoteAmount) * 100.0 / float64(epochVote.attesterDuties.validatorBalance)

	t.logger.Infof("Epoch %v votes [%v]", epoch, epochVote.headRoot.String())
	t.logger.Infof("epoch %v validators: %v (eff. balance: %v)", epoch, epochVote.attesterDuties.validatorCount, epochVote.attesterDuties.validatorBalance)
	t.logger.Infof("epoch %v target votes: %v (%.2f%%)", epoch, epochVote.currentEpoch.targetVoteCount+epochVote.nextEpoch.targetVoteCount, targetPercent)
	t.logger.Infof("epoch %v head votes: %v (%.2f%%)", epoch, epochVote.currentEpoch.headVoteCount+epochVote.nextEpoch.headVoteCount, headPercent)
	t.logger.Infof("epoch %v total votes: %v (%.2f%%)", epoch, epochVote.currentEpoch.totalVoteCount+epochVote.nextEpoch.totalVoteCount, totalPercent)

	t.ctx.Outputs.SetVar("lastCheckedEpoch", epoch)
	t.ctx.Outputs.SetVar("validatorCount", epochVote.attesterDuties.validatorCount)
	t.ctx.Outputs.SetVar("validatorBalance", epochVote.attesterDuties.validatorBalance)
	t.ctx.Outputs.SetVar("targetVotes", epochVote.currentEpoch.targetVoteCount+epochVote.nextEpoch.targetVoteCount)
	t.ctx.Outputs.SetVar("targetVotesPercent", targetPercent)
	t.ctx.Outputs.SetVar("headVotes", epochVote.currentEpoch.headVoteCount+epochVote.nextEpoch.headVoteCount)
	t.ctx.Outputs.SetVar("headVotesPercent", headPercent)
	t.ctx.Outputs.SetVar("totalVotes", epochVote.currentEpoch.totalVoteCount+epochVote.nextEpoch.totalVoteCount)
	t.ctx.Outputs.SetVar("totalVotesPercent", totalPercent)

	if t.config.MinTargetPercent > 0 && targetPercent < float64(t.config.MinTargetPercent) {
		t.logger.Debugf("check failed for epoch %v: target vote percent (want: >= %v, have: %.2f%)", epoch, t.config.MinTargetPercent, targetPercent)
		return false
	}

	if t.config.MaxTargetPercent < 100 && targetPercent > float64(t.config.MaxTargetPercent) {
		t.logger.Debugf("check failed for epoch %v: target vote percent (want: <= %v, have: %.2f%)", epoch, t.config.MaxTargetPercent, targetPercent)
		return false
	}

	if t.config.MinHeadPercent > 0 && headPercent < float64(t.config.MinHeadPercent) {
		t.logger.Debugf("check failed for epoch %v: head vote percent (want: >= %v, have: %.2f%)", epoch, t.config.MinHeadPercent, headPercent)
		return false
	}

	if t.config.MaxHeadPercent < 100 && headPercent > float64(t.config.MaxHeadPercent) {
		t.logger.Debugf("check failed for epoch %v: head vote percent (want: <= %v, have: %.2f%)", epoch, t.config.MaxHeadPercent, headPercent)
		return false
	}

	if t.config.MinTotalPercent > 0 && totalPercent < float64(t.config.MinTotalPercent) {
		t.logger.Debugf("check failed for epoch %v: total vote percent (want: >= %v, have: %.2f%)", epoch, t.config.MinTotalPercent, totalPercent)
		return false
	}

	if t.config.MaxTotalPercent < 100 && totalPercent > float64(t.config.MaxTotalPercent) {
		t.logger.Debugf("check failed for epoch %v: total vote percent (want: <= %v, have: %.2f%)", epoch, t.config.MaxTotalPercent, totalPercent)
		return false
	}

	return true
}

type epochVotes struct {
	headRoot       phase0.Root
	dependentRoot  phase0.Root
	targetRoot     phase0.Root
	attesterDuties *attesterDuties
	currentEpoch   struct {
		targetVoteAmount uint64
		targetVoteCount  uint64
		headVoteAmount   uint64
		headVoteCount    uint64
		totalVoteAmount  uint64
		totalVoteCount   uint64
	}
	nextEpoch struct {
		targetVoteAmount uint64
		targetVoteCount  uint64
		headVoteAmount   uint64
		headVoteCount    uint64
		totalVoteAmount  uint64
		totalVoteCount   uint64
	}
	activityMap map[uint64]bool
}

func (t *Task) newEpochVotes(base *epochVotes) *epochVotes {
	votes := &epochVotes{
		activityMap: map[uint64]bool{},
	}

	if base != nil {
		votes.dependentRoot = base.dependentRoot
		votes.targetRoot = base.targetRoot
		votes.attesterDuties = base.attesterDuties
		votes.currentEpoch = base.currentEpoch
		votes.nextEpoch = base.nextEpoch

		for i, b := range base.activityMap {
			votes.activityMap[i] = b
		}
	}

	return votes
}

func (t *Task) aggregateEpochVotes(ctx context.Context, epoch uint64) []*epochVotes {
	t1 := time.Now()

	consensusBlockCache := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache()
	specs := consensusBlockCache.GetSpecs()

	firstSlot := epoch * specs.SlotsPerEpoch
	lastSlot := firstSlot + (2 * specs.SlotsPerEpoch)

	allHeads := map[phase0.Root]bool{}
	allVotes := map[phase0.Root]*epochVotes{}

	for slot := firstSlot; slot <= lastSlot; slot++ {
		for _, block := range consensusBlockCache.GetCachedBlocksBySlot(phase0.Slot(slot)) {
			blockBody := block.AwaitBlock(ctx, 500*time.Millisecond)
			if blockBody == nil {
				continue
			}

			parentRoot := block.GetParentRoot()
			if parentRoot == nil {
				continue
			}

			parentVote, isOk := allVotes[*parentRoot]
			votes := t.newEpochVotes(parentVote)
			votes.headRoot = block.Root

			if !isOk {
				votes.dependentRoot = *parentRoot

				votes.attesterDuties = t.attesterDutyMap[epoch][votes.dependentRoot]
				if votes.attesterDuties == nil {
					t.logger.Warnf("cannot find attestor duties for epoch %v / root %v", epoch, votes.dependentRoot.String())
					continue
				}

				if slot == firstSlot {
					votes.targetRoot = block.Root
				} else {
					votes.targetRoot = *parentRoot
				}
			} else if allHeads[*parentRoot] {
				delete(allHeads, *parentRoot)
			}

			allHeads[block.Root] = true
			allVotes[block.Root] = votes

			isNextEpoch := slot-firstSlot >= specs.SlotsPerEpoch

			attestationsVersioned, err := blockBody.Attestations()
			if err != nil {
				continue
			}

			for attIdx, att := range attestationsVersioned {
				attData, err1 := att.Data()
				if err1 != nil {
					continue
				}

				if uint64(attData.Slot)/specs.SlotsPerEpoch != epoch {
					continue
				}

				attAggregationBits, err := att.AggregationBits()
				if err != nil {
					continue
				}

				voteAmount := uint64(0)
				voteCount := uint64(0)

				if att.Version >= spec.DataVersionElectra {
					// EIP-7549 changes the attestation aggregation
					// there can now be attestations from all committees aggregated into a single attestation aggregate
					committeeBits, err := att.CommitteeBits()
					if err != nil {
						t.logger.Debugf("aggregateEpochVotes slot %v failed, can't get committeeBits for attestation %v: %v", slot, attIdx, err)
						continue
					}

					aggregationBitsOffset := uint64(0)

					for committee := uint64(0); committee < specs.MaxCommitteesPerSlot; committee++ {
						if !committeeBits.BitAt(committee) {
							continue
						}

						voteAmt, voteCnt, committeeSize := t.aggregateAttestationVotes(votes, uint64(attData.Slot), committee, attAggregationBits, 0)
						voteAmount += voteAmt
						voteCount += voteCnt
						aggregationBitsOffset += committeeSize
					}
				} else {
					// pre electra attestation aggregation
					voteAmt, voteCnt, _ := t.aggregateAttestationVotes(votes, uint64(attData.Slot), uint64(attData.Index), attAggregationBits, 0)
					voteAmount += voteAmt
					voteCount += voteCnt
				}

				if bytes.Equal(attData.Target.Root[:], votes.targetRoot[:]) {
					if isNextEpoch {
						votes.nextEpoch.targetVoteCount += voteCount
						votes.nextEpoch.targetVoteAmount += voteAmount
					} else {
						votes.currentEpoch.targetVoteCount += voteCount
						votes.currentEpoch.targetVoteAmount += voteAmount
					}
				}

				if bytes.Equal(attData.BeaconBlockRoot[:], parentRoot[:]) {
					if isNextEpoch {
						votes.nextEpoch.headVoteCount += voteCount
						votes.nextEpoch.headVoteAmount += voteAmount
					} else {
						votes.currentEpoch.headVoteCount += voteCount
						votes.currentEpoch.headVoteAmount += voteAmount
					}
				}

				if isNextEpoch {
					votes.nextEpoch.totalVoteCount += voteCount
					votes.nextEpoch.totalVoteAmount += voteAmount
				} else {
					votes.currentEpoch.totalVoteCount += voteCount
					votes.currentEpoch.totalVoteAmount += voteAmount
				}
			}
		}
	}

	votes := []*epochVotes{}
	for root := range allHeads {
		votes = append(votes, allVotes[root])
	}

	t.logger.Debugf("aggregated epoch %v votes in %v", epoch, time.Since(t1))

	return votes
}

func (t *Task) aggregateAttestationVotes(votes *epochVotes, slot, committee uint64, aggregationBits bitfield.Bitfield, aggregationBitsOffset uint64) (voteAmount, voteCount, validatorCount uint64) {
	attKey := fmt.Sprintf("%v-%v", slot, committee)
	voteValidators := votes.attesterDuties.duties[attKey]

	for bitIdx, attDuty := range voteValidators {
		validatorIdx := attDuty.validator
		if aggregationBits.BitAt(uint64(bitIdx) + aggregationBitsOffset) { //nolint:gosec // no overflow possible
			if votes.activityMap[validatorIdx] {
				continue
			}

			voteAmount += attDuty.balance
			voteCount++
			votes.activityMap[validatorIdx] = true
		}
	}

	validatorCount = uint64(len(voteValidators))

	return
}
