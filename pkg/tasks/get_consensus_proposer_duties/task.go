// Package getconsensusproposerduties fetches the proposer schedule
// for one epoch and exposes the duties array plus a couple of useful
// derived fields (first future-slot duty + a deduped list of
// validator indices) as task outputs.
//
// This is a small, generic primitive — playbooks compose it with
// `configVars` to craft realistic inputs for endpoints that want a
// known proposer slot or a list of real validator indices.
package getconsensusproposerduties

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus/rpc"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

const (
	defaultRequestTimeout = 15 * time.Second
	slotsPerEpochFallback = 32
)

var (
	TaskName       = "get_consensus_proposer_duties"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Fetch proposer duties for one epoch and emit the duties array plus convenience fields (first future-slot proposer, list of unique validator indices).",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{Name: "epoch", Type: "int", Description: "The epoch whose duties were fetched."},
			{Name: "duties", Type: "array", Description: "Array of {slot, validator_index, pubkey} objects in slot order."},
			{Name: "firstFutureSlot", Type: "int", Description: "Slot of the first duty strictly after the current head slot, or 0 if no future duty in this epoch."},
			{Name: "firstFutureValidatorIndex", Type: "int", Description: "Validator index of firstFutureSlot's proposer."},
			{Name: "validatorIndices", Type: "array", Description: "Up to maxDuties unique validator indices drawn from this epoch's schedule."},
		},
		NewTask: NewTask,
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

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	if err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars); err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	if config.RequestTimeout.Duration == 0 {
		config.RequestTimeout = helper.Duration{Duration: defaultRequestTimeout}
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	client, err := t.pickClient()
	if err != nil {
		return err
	}

	rpcClient := client.ConsensusClient.GetRPCClient()

	// Resolve "current epoch" lazily so the user can give either an
	// absolute Epoch or just an EpochOffset.
	targetEpoch, headSlot, err := t.resolveTargetEpoch(ctx, rpcClient)
	if err != nil {
		return err
	}

	dCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)

	duties, err := rpcClient.GetProposerDuties(dCtx, targetEpoch)

	cancel()

	if err != nil {
		return fmt.Errorf("fetch proposer duties for epoch %d: %w", targetEpoch, err)
	}

	// Build the outputs.
	dutiesOut := make([]map[string]any, 0, len(duties))
	validatorIndices := make([]uint64, 0, t.config.MaxDuties)
	seenIdx := map[uint64]bool{}

	var (
		firstFutureSlot uint64
		firstFutureIdx  uint64
	)

	for _, d := range duties {
		if d == nil {
			continue
		}

		slot := uint64(d.Slot)
		idx := uint64(d.ValidatorIndex)

		dutiesOut = append(dutiesOut, map[string]any{
			"slot":            slot,
			"validator_index": idx,
			"pubkey":          fmt.Sprintf("%#x", d.PubKey),
		})

		if firstFutureSlot == 0 && slot > headSlot {
			firstFutureSlot = slot
			firstFutureIdx = idx
		}

		if !seenIdx[idx] && len(validatorIndices) < t.config.MaxDuties {
			validatorIndices = append(validatorIndices, idx)
			seenIdx[idx] = true
		}
	}

	t.ctx.Outputs.SetVar("epoch", targetEpoch)
	t.ctx.Outputs.SetVar("duties", dutiesOut)
	t.ctx.Outputs.SetVar("firstFutureSlot", firstFutureSlot)
	t.ctx.Outputs.SetVar("firstFutureValidatorIndex", firstFutureIdx)
	t.ctx.Outputs.SetVar("validatorIndices", validatorIndices)

	t.logger.WithFields(logrus.Fields{
		"epoch":           targetEpoch,
		"duties":          len(dutiesOut),
		"firstFutureSlot": firstFutureSlot,
	}).Info("proposer duties fetched")

	t.ctx.ReportProgress(100, "duties fetched")

	return nil
}

func (t *Task) pickClient() (*clients.PoolClient, error) {
	pool := t.ctx.Scheduler.GetServices().ClientPool()
	matching := pool.GetClientsByNamePatterns(t.config.ClientPattern, "")

	for _, c := range matching {
		if c.ConsensusClient != nil && c.ConsensusClient.GetStatus() == consensus.ClientStatusOnline {
			return c, nil
		}
	}

	if len(matching) > 0 {
		return matching[0], nil
	}

	if t.config.ClientPattern != "" {
		return nil, fmt.Errorf("no consensus client matches clientPattern %q", t.config.ClientPattern)
	}

	return nil, fmt.Errorf("no consensus clients available")
}

// resolveTargetEpoch returns (targetEpoch, headSlot, error). When the
// user supplied an explicit Epoch we use it directly; otherwise we
// query the head and apply EpochOffset.
func (t *Task) resolveTargetEpoch(ctx context.Context, rpcClient *rpc.BeaconClient) (targetEpoch, headSlot uint64, err error) {
	if t.config.Epoch > 0 {
		// Even when the epoch is explicit we still need the head slot
		// for "firstFutureSlot" — best effort, ignore errors.
		headCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)
		head, _ := rpcClient.GetLatestBlockHead(headCtx)

		cancel()

		if head != nil {
			headSlot = uint64(head.Header.Message.Slot)
		}

		targetEpoch = t.config.Epoch

		return targetEpoch, headSlot, nil
	}

	headCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)
	head, headErr := rpcClient.GetLatestBlockHead(headCtx)

	cancel()

	if headErr != nil || head == nil {
		return 0, 0, fmt.Errorf("fetch head: %w", headErr)
	}

	slotsPerEpoch := t.fetchSlotsPerEpoch(ctx, rpcClient)
	headSlot = uint64(head.Header.Message.Slot)
	currentEpoch := headSlot / slotsPerEpoch

	targetEpoch = currentEpoch

	if t.config.EpochOffset > 0 {
		targetEpoch = currentEpoch + uint64(t.config.EpochOffset)
	}

	return targetEpoch, headSlot, nil
}

func (t *Task) fetchSlotsPerEpoch(ctx context.Context, rpcClient *rpc.BeaconClient) uint64 {
	sCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)
	defer cancel()

	specs, err := rpcClient.GetConfigSpecs(sCtx)
	if err != nil || specs == nil {
		return slotsPerEpochFallback
	}

	if v, ok := specs["SLOTS_PER_EPOCH"]; ok {
		if vu, ok := v.(uint64); ok && vu > 0 {
			return vu
		}
	}

	return slotsPerEpochFallback
}
