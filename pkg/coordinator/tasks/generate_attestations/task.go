package generateattestations

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus/rpc"
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
	TaskName       = "generate_attestations"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates valid attestations and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger

	valSeed       []byte
	validatorKeys map[phase0.ValidatorIndex]*validatorKey

	// Cache for committee duties per epoch
	dutiesCache map[uint64][]*v1.BeaconCommittee
}

type validatorKey struct {
	privkey *e2types.BLSPrivateKey
	pubkey  []byte
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

	t.valSeed, err = t.mnemonicToSeed(config.Mnemonic)
	if err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	// Initialize validator keys
	err := t.initValidatorKeys()
	if err != nil {
		return err
	}

	if len(t.validatorKeys) == 0 {
		return fmt.Errorf("no validators found for given key range")
	}

	t.logger.Infof("found %d validators for key range", len(t.validatorKeys))

	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	consensusPool.GetBlockCache().SetMinFollowDistance(uint64(1000))

	// Subscribe to epoch events
	epochSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer epochSubscription.Unsubscribe()

	// Subscribe to slot events for timing
	slotSubscription := consensusPool.GetBlockCache().SubscribeWallclockSlotEvent(10)
	defer slotSubscription.Unsubscribe()

	// Get current epoch to start from
	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return fmt.Errorf("failed to get current wallclock: %w", err)
	}

	startEpoch := currentEpoch.Number()
	if t.config.LastEpochAttestations && startEpoch > 0 {
		startEpoch--
	}

	t.logger.Infof("starting attestation generation from epoch %d", startEpoch)

	specs := consensusPool.GetBlockCache().GetSpecs()
	totalAttestations := 0
	processedEpochs := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case slot := <-slotSubscription.Channel():
			// Skip slot processing in sendAllLastEpoch mode
			if t.config.SendAllLastEpoch {
				continue
			}

			// Process attestations for this slot
			count, err := t.processSlot(ctx, slot.Number(), startEpoch)
			if err != nil {
				t.logger.Warnf("error processing slot %d: %v", slot.Number(), err)
				continue
			}

			if count > 0 {
				totalAttestations += count
				t.ctx.SetResult(types.TaskResultSuccess)
				t.logger.Infof("sent %d attestations for slot %d (total: %d)", count, slot.Number(), totalAttestations)
			}

			// Check limits
			if t.config.LimitTotal > 0 && totalAttestations >= t.config.LimitTotal {
				t.logger.Infof("reached total attestation limit: %d", totalAttestations)
				return nil
			}

		case epoch := <-epochSubscription.Channel():
			epochNum := epoch.Number()
			if epochNum <= startEpoch {
				continue
			}

			if t.config.SendAllLastEpoch {
				// Process all slots from the previous epoch at once
				prevEpoch := epochNum - 1
				t.logger.Infof("processing all attestations for epoch %d", prevEpoch)

				epochAttestations := 0
				for slotOffset := uint64(0); slotOffset < specs.SlotsPerEpoch; slotOffset++ {
					targetSlot := prevEpoch*specs.SlotsPerEpoch + slotOffset

					count, err := t.processSlotForEpoch(ctx, targetSlot, prevEpoch)
					if err != nil {
						t.logger.Warnf("error processing slot %d: %v", targetSlot, err)
						continue
					}

					epochAttestations += count
					totalAttestations += count

					// Check total limit
					if t.config.LimitTotal > 0 && totalAttestations >= t.config.LimitTotal {
						t.logger.Infof("reached total attestation limit: %d", totalAttestations)
						t.ctx.SetResult(types.TaskResultSuccess)
						return nil
					}
				}

				if epochAttestations > 0 {
					t.ctx.SetResult(types.TaskResultSuccess)
					t.logger.Infof("sent %d attestations for epoch %d (total: %d)", epochAttestations, prevEpoch, totalAttestations)
				}
			}

			processedEpochs++
			t.logger.Infof("completed epoch %d, processed epochs: %d", epochNum-1, processedEpochs)

			// Check epoch limit
			if t.config.LimitEpochs > 0 && processedEpochs >= t.config.LimitEpochs {
				t.logger.Infof("reached epoch limit: %d", processedEpochs)
				return nil
			}
		}
	}
}

func (t *Task) initValidatorKeys() error {
	t.validatorKeys = make(map[phase0.ValidatorIndex]*validatorKey)

	validators := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()
	if validators == nil {
		return fmt.Errorf("failed to get validator set")
	}

	startIndex := uint64(0)
	if t.config.StartIndex > 0 {
		startIndex = uint64(t.config.StartIndex) //nolint:gosec // no overflow possible
	}

	endIndex := startIndex + uint64(t.config.IndexCount) //nolint:gosec // no overflow possible

	for accountIdx := startIndex; accountIdx < endIndex; accountIdx++ {
		validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

		validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.valSeed, validatorKeyPath)
		if err != nil {
			return fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
		}

		validatorPubkey := validatorPrivkey.PublicKey().Marshal()

		// Find this validator in the validator set
		for valIdx, val := range validators {
			if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
				if val.Status != v1.ValidatorStateActiveOngoing && val.Status != v1.ValidatorStateActiveExiting {
					t.logger.Debugf("validator %d is not active (status: %s), skipping", valIdx, val.Status)
					continue
				}

				t.validatorKeys[valIdx] = &validatorKey{
					privkey: validatorPrivkey,
					pubkey:  validatorPubkey,
				}

				break
			}
		}
	}

	return nil
}

func (t *Task) processSlot(ctx context.Context, slot uint64, startEpoch uint64) (int, error) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	specs := consensusPool.GetBlockCache().GetSpecs()

	slotEpoch := slot / specs.SlotsPerEpoch

	// For last epoch attestations, we attest for the previous epoch
	targetEpoch := slotEpoch
	if t.config.LastEpochAttestations && slotEpoch > 0 {
		targetEpoch = slotEpoch - 1
	}

	// Skip if before start epoch
	if targetEpoch < startEpoch {
		return 0, nil
	}

	// Get the target slot to attest for
	targetSlot := slot
	if t.config.LastEpochAttestations {
		// Attest for the same relative slot in the previous epoch
		slotInEpoch := slot % specs.SlotsPerEpoch
		targetSlot = targetEpoch*specs.SlotsPerEpoch + slotInEpoch
	}

	// Get committee duties for the target epoch
	duties, err := t.getCommitteeDuties(ctx, targetEpoch)
	if err != nil {
		return 0, fmt.Errorf("failed to get committee duties: %w", err)
	}

	// Find our validators' duties for this slot
	slotDuties := t.findSlotDuties(targetSlot, duties)
	if len(slotDuties) == 0 {
		return 0, nil
	}

	// Group duties by committee index
	committeeGroups := make(map[phase0.CommitteeIndex][]*validatorDuty)
	for _, duty := range slotDuties {
		committeeGroups[duty.committeeIndex] = append(committeeGroups[duty.committeeIndex], duty)
	}

	count := 0

	// Get attestation data and submit for each committee
	// Attestation data is requested for targetSlot, then lateHead/sourceOffset/targetOffset are applied on top
	for committeeIdx, committeeDuties := range committeeGroups {
		submitted, err := t.generateAndSubmitAttestation(ctx, targetSlot, committeeIdx, committeeDuties)
		if err != nil {
			t.logger.Warnf("failed to submit attestation for slot %d committee %d: %v", targetSlot, committeeIdx, err)
			continue
		}

		count += submitted
	}

	return count, nil
}

// processSlotForEpoch processes attestations for a specific slot in a given epoch.
// This is used by sendAllLastEpoch mode where we know the target epoch directly.
func (t *Task) processSlotForEpoch(ctx context.Context, slot uint64, epoch uint64) (int, error) {
	// Get committee duties for the epoch
	duties, err := t.getCommitteeDuties(ctx, epoch)
	if err != nil {
		return 0, fmt.Errorf("failed to get committee duties: %w", err)
	}

	// Find our validators' duties for this slot
	slotDuties := t.findSlotDuties(slot, duties)
	if len(slotDuties) == 0 {
		return 0, nil
	}

	// Group duties by committee index
	committeeGroups := make(map[phase0.CommitteeIndex][]*validatorDuty)
	for _, duty := range slotDuties {
		committeeGroups[duty.committeeIndex] = append(committeeGroups[duty.committeeIndex], duty)
	}

	count := 0

	// Get attestation data and submit for each committee
	for committeeIdx, committeeDuties := range committeeGroups {
		submitted, err := t.generateAndSubmitAttestation(ctx, slot, committeeIdx, committeeDuties)
		if err != nil {
			t.logger.Warnf("failed to submit attestation for slot %d committee %d: %v", slot, committeeIdx, err)
			continue
		}

		count += submitted
	}

	return count, nil
}

type validatorDuty struct {
	validatorIndex      phase0.ValidatorIndex
	committeeIndex      phase0.CommitteeIndex
	committeeLength     uint64
	positionInCommittee uint64
}

func (t *Task) findSlotDuties(slot uint64, duties []*v1.BeaconCommittee) []*validatorDuty {
	var result []*validatorDuty

	for _, committee := range duties {
		if uint64(committee.Slot) != slot {
			continue
		}

		for position, valIdx := range committee.Validators {
			if _, ok := t.validatorKeys[valIdx]; ok {
				result = append(result, &validatorDuty{
					validatorIndex:      valIdx,
					committeeIndex:      committee.Index,
					committeeLength:     uint64(len(committee.Validators)),
					positionInCommittee: uint64(position),
				})
			}
		}
	}

	return result
}

func (t *Task) getCommitteeDuties(ctx context.Context, epoch uint64) ([]*v1.BeaconCommittee, error) {
	// Check cache first
	if duties, ok := t.dutiesCache[epoch]; ok {
		return duties, nil
	}

	client := t.getClient()
	if client == nil {
		return nil, fmt.Errorf("no client available")
	}

	duties, err := client.GetRPCClient().GetCommitteeDuties(ctx, "head", epoch)
	if err != nil {
		return nil, err
	}

	// Initialize cache if needed
	if t.dutiesCache == nil {
		t.dutiesCache = make(map[uint64][]*v1.BeaconCommittee)
	}

	// Clean up old epochs from cache (keep only current and previous)
	for cachedEpoch := range t.dutiesCache {
		if cachedEpoch+2 < epoch {
			delete(t.dutiesCache, cachedEpoch)
		}
	}

	// Store in cache
	t.dutiesCache[epoch] = duties

	return duties, nil
}

func (t *Task) generateAndSubmitAttestation(ctx context.Context, slot uint64, committeeIdx phase0.CommitteeIndex, duties []*validatorDuty) (int, error) {
	clients := t.getClients()
	if len(clients) == 0 {
		return 0, fmt.Errorf("no client available")
	}

	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	specs := consensusPool.GetBlockCache().GetSpecs()
	genesis := consensusPool.GetBlockCache().GetGenesis()

	// Get attestation data from beacon node, retry with different clients if needed
	var attData *phase0.AttestationData
	var lastErr error
	for _, client := range clients {
		var err error
		attData, err = client.GetRPCClient().GetAttestationData(ctx, slot, uint64(committeeIdx))
		if err == nil {
			break
		}
		lastErr = err
		t.logger.Debugf("failed to get attestation data from %s: %v, trying next client", client.GetName(), err)
	}
	if attData == nil {
		return 0, fmt.Errorf("failed to get attestation data from all clients: %w", lastErr)
	}

	// Use first client for remaining operations
	client := clients[0]

	// Apply static late head offset if configured (applies to all attestations in this batch)
	if t.config.LateHead != 0 {
		modifiedData := t.applyLateHead(attData, t.config.LateHead)
		attData = modifiedData
	}

	// Get fork state
	forkState, err := client.GetRPCClient().GetForkState(ctx, "head")
	if err != nil {
		return 0, fmt.Errorf("failed to get fork state: %w", err)
	}

	// Compute the signing domain
	epoch := uint64(attData.Slot) / specs.SlotsPerEpoch
	forkVersion := forkState.CurrentVersion
	if epoch < uint64(forkState.Epoch) {
		forkVersion = forkState.PreviousVersion
	}

	dom := common.ComputeDomain(common.DOMAIN_BEACON_ATTESTER, common.Version(forkVersion), tree.Root(genesis.GenesisValidatorsRoot))

	if len(duties) == 0 {
		return 0, fmt.Errorf("no duties provided")
	}

	// Parse random late head config
	randomMin, randomMax, randomEnabled, _ := t.config.ParseRandomLateHead()
	clusterSize := t.config.LateHeadClusterSize
	if clusterSize <= 0 {
		clusterSize = 1 // Default: each attestation gets its own random offset
	}

	// Create SingleAttestation objects for each validator (Electra format)
	var singleAttestations []*rpc.SingleAttestation

	var currentClusterOffset int
	var clusterAttData *phase0.AttestationData
	attestationCount := 0

	for _, duty := range duties {
		valKey := t.validatorKeys[duty.validatorIndex]
		if valKey == nil {
			continue
		}

		// Apply per-attestation or per-cluster random late head if configured
		attDataForValidator := attData
		if randomEnabled {
			// Generate new random offset at start or when cluster is full
			if attestationCount%clusterSize == 0 {
				currentClusterOffset = randomMin + rand.Intn(randomMax-randomMin+1)
				if currentClusterOffset != 0 {
					clusterAttData = t.applyLateHead(attData, currentClusterOffset)
				} else {
					clusterAttData = attData
				}
			}
			attDataForValidator = clusterAttData
			attestationCount++
		}

		// Sign attestation data
		msgRoot, err := attDataForValidator.HashTreeRoot()
		if err != nil {
			return 0, fmt.Errorf("failed to hash attestation data: %w", err)
		}

		signingRoot := common.ComputeSigningRoot(msgRoot, dom)

		var secKey hbls.SecretKey
		if err := secKey.Deserialize(valKey.privkey.Marshal()); err != nil {
			return 0, fmt.Errorf("failed to deserialize private key: %w", err)
		}

		sig := secKey.SignHash(signingRoot[:])

		singleAtt := &rpc.SingleAttestation{
			CommitteeIndex: uint64(committeeIdx),
			AttesterIndex:  uint64(duty.validatorIndex),
			Data:           attDataForValidator,
			Signature:      fmt.Sprintf("0x%x", sig.Serialize()),
		}
		singleAttestations = append(singleAttestations, singleAtt)
	}

	if len(singleAttestations) == 0 {
		return 0, fmt.Errorf("no attestations generated")
	}

	// Submit attestations
	err = client.GetRPCClient().SubmitAttestations(ctx, singleAttestations)
	if err != nil {
		return 0, fmt.Errorf("failed to submit attestation: %w", err)
	}

	return len(singleAttestations), nil
}

// applyLateHead applies a late head offset to the attestation data.
// Positive offset goes back (older blocks), negative goes forward.
// If the offset would result in a head before the target epoch, the head is clamped
// to the target epoch's first slot (using the target root).
func (t *Task) applyLateHead(attData *phase0.AttestationData, offset int) *phase0.AttestationData {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	specs := consensusPool.GetBlockCache().GetSpecs()

	newRoot, newSlot := t.walkBlocks(attData.BeaconBlockRoot, uint64(attData.Slot), offset)

	// Validate: head slot must be >= target epoch's first slot
	// If not, clamp to target epoch (use target root as head)
	targetEpochFirstSlot := uint64(attData.Target.Epoch) * specs.SlotsPerEpoch
	if newSlot < targetEpochFirstSlot {
		t.logger.Debugf("late head offset %d would result in invalid head (slot %d < target epoch slot %d), clamping to target",
			offset, newSlot, targetEpochFirstSlot)
		newRoot = attData.Target.Root
		newSlot = targetEpochFirstSlot
	}

	modifiedData := &phase0.AttestationData{
		Slot:            attData.Slot,
		Index:           attData.Index,
		BeaconBlockRoot: newRoot,
		Source: &phase0.Checkpoint{
			Epoch: attData.Source.Epoch,
			Root:  attData.Source.Root,
		},
		Target: &phase0.Checkpoint{
			Epoch: attData.Target.Epoch,
			Root:  attData.Target.Root,
		},
	}

	t.logger.Debugf("late head offset: %d, new slot: %d [root: %x]", offset, newSlot, newRoot)

	return modifiedData
}

// walkBlocks walks N blocks from the given root.
// Positive steps go backwards (using parentRoot), negative steps go forward (finding child blocks).
// Returns the resulting root and slot. Always returns a valid slot from the last known block.
func (t *Task) walkBlocks(startRoot phase0.Root, startSlot uint64, steps int) (phase0.Root, uint64) {
	if steps > 0 {
		return t.walkBackBlocks(startRoot, startSlot, steps)
	} else if steps < 0 {
		return t.walkForwardBlocks(startRoot, startSlot, -steps)
	}

	// steps == 0, try to get actual slot from block, fallback to startSlot
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	blockCache := consensusPool.GetBlockCache()

	block := blockCache.GetCachedBlockByRoot(startRoot)
	if block != nil {
		return startRoot, uint64(block.Slot)
	}

	return startRoot, startSlot
}

// walkBackBlocks walks back N blocks from the given root using parentRoot.
func (t *Task) walkBackBlocks(startRoot phase0.Root, startSlot uint64, steps int) (phase0.Root, uint64) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	blockCache := consensusPool.GetBlockCache()

	currentRoot := startRoot
	currentSlot := startSlot

	// Get initial block to determine starting slot
	block := blockCache.GetCachedBlockByRoot(currentRoot)
	if block != nil {
		currentSlot = uint64(block.Slot)
	}

	for range steps {
		block := blockCache.GetCachedBlockByRoot(currentRoot)
		if block == nil {
			break
		}

		currentSlot = uint64(block.Slot)

		parentRoot := block.GetParentRoot()
		if parentRoot == nil {
			break
		}

		currentRoot = *parentRoot
	}

	// Get the final slot
	if block := blockCache.GetCachedBlockByRoot(currentRoot); block != nil {
		currentSlot = uint64(block.Slot)
	}

	return currentRoot, currentSlot
}

// walkForwardBlocks walks forward N blocks from the given root by finding child blocks.
func (t *Task) walkForwardBlocks(startRoot phase0.Root, startSlot uint64, steps int) (phase0.Root, uint64) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	blockCache := consensusPool.GetBlockCache()

	currentRoot := startRoot
	currentSlot := startSlot

	// Get initial block to determine starting slot
	block := blockCache.GetCachedBlockByRoot(currentRoot)
	if block != nil {
		currentSlot = uint64(block.Slot)
	}

	for range steps {
		// Find a child block whose parent is currentRoot
		childBlock := t.findChildBlock(currentRoot)
		if childBlock == nil {
			break
		}

		currentRoot = childBlock.Root
		currentSlot = uint64(childBlock.Slot)
	}

	return currentRoot, currentSlot
}

// findChildBlock finds a cached block whose parent is the given root.
func (t *Task) findChildBlock(parentRoot phase0.Root) *consensus.Block {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	blockCache := consensusPool.GetBlockCache()

	// Get the parent block to know its slot
	parentBlock := blockCache.GetCachedBlockByRoot(parentRoot)
	if parentBlock == nil {
		return nil
	}

	// Search in slots after the parent
	parentSlot := uint64(parentBlock.Slot)
	for searchSlot := parentSlot + 1; searchSlot <= parentSlot+32; searchSlot++ {
		blocks := blockCache.GetCachedBlocksBySlot(phase0.Slot(searchSlot))
		for _, block := range blocks {
			blockParent := block.GetParentRoot()
			if blockParent != nil && *blockParent == parentRoot {
				return block
			}
		}
	}

	return nil
}

func (t *Task) getClient() *consensus.Client {
	clients := t.getClients()
	if len(clients) == 0 {
		return nil
	}

	return clients[0]
}

func (t *Task) getClients() []*consensus.Client {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	consensusPool := clientPool.GetConsensusPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		allClients := consensusPool.GetAllEndpoints()
		clients := make([]*consensus.Client, 0, len(allClients))
		for _, c := range allClients {
			if consensusPool.IsClientReady(c) {
				clients = append(clients, c)
			}
		}

		return clients
	}

	poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
	clients := make([]*consensus.Client, 0, len(poolClients))
	for _, c := range poolClients {
		if c.ConsensusClient != nil && consensusPool.IsClientReady(c.ConsensusClient) {
			clients = append(clients, c.ConsensusClient)
		}
	}

	return clients
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
