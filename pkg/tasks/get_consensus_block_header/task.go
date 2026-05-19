// Package getconsensusblockheader fetches one beacon-block header and
// surfaces its slot / root / proposer-index / parent_root / state_root
// as task outputs.
//
// It's intentionally small: a single beacon API call against one
// chosen CL client, returning the result. Playbooks compose these
// little reads with `configVars` to feed realistic inputs into other
// tasks (e.g. the GLOAS API compatibility matrix uses it to find a
// canonical slot/root pair to query envelope endpoints against).
package getconsensusblockheader

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus/rpc"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
	v1 "github.com/ethpandaops/go-eth2-client/api/v1"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

const defaultRequestTimeout = 15 * time.Second

var (
	TaskName       = "get_consensus_block_header"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Fetch a beacon-block header (by slot, by root, or head minus an offset) and emit its identifying fields as task outputs.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{Name: "slot", Type: "int", Description: "Slot of the returned block."},
			{Name: "root", Type: "string", Description: "Canonical block root (0x-prefixed hex)."},
			{Name: "proposerIndex", Type: "int", Description: "Validator index of the block's proposer."},
			{Name: "parentRoot", Type: "string", Description: "Parent block root."},
			{Name: "stateRoot", Type: "string", Description: "State root committed in this block."},
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

	header, err := t.fetchHeader(ctx, rpcClient)
	if err != nil {
		return err
	}

	slot := uint64(header.Header.Message.Slot)
	root := fmt.Sprintf("%#x", header.Root)

	t.ctx.Outputs.SetVar("slot", slot)
	t.ctx.Outputs.SetVar("root", root)
	t.ctx.Outputs.SetVar("proposerIndex", uint64(header.Header.Message.ProposerIndex))
	t.ctx.Outputs.SetVar("parentRoot", fmt.Sprintf("%#x", header.Header.Message.ParentRoot))
	t.ctx.Outputs.SetVar("stateRoot", fmt.Sprintf("%#x", header.Header.Message.StateRoot))

	t.logger.WithFields(logrus.Fields{
		"slot": slot,
		"root": root,
	}).Info("block header fetched")

	t.ctx.ReportProgress(100, "header fetched")

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

// fetchHeader dispatches to the appropriate RPC method based on
// config priority: explicit BlockRoot wins over Slot, which wins over
// the default "head minus HeadOffset" walk.
func (t *Task) fetchHeader(ctx context.Context, rpcClient *rpc.BeaconClient) (*v1.BeaconBlockHeader, error) {
	if t.config.BlockRoot != "" {
		root, err := parseRoot(t.config.BlockRoot)
		if err != nil {
			return nil, err
		}

		hCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)
		defer cancel()

		raw, err := rpcClient.GetBlockHeaderByBlockroot(hCtx, root)
		if err != nil {
			return nil, fmt.Errorf("fetch header by root %s: %w", t.config.BlockRoot, err)
		}

		return raw, nil
	}

	if t.config.Slot > 0 {
		return t.fetchHeaderBySlot(ctx, rpcClient, t.config.Slot)
	}

	headCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)
	head, err := rpcClient.GetLatestBlockHead(headCtx)

	cancel()

	if err != nil || head == nil {
		return nil, fmt.Errorf("fetch head: %w", err)
	}

	if t.config.HeadOffset <= 0 {
		return head, nil
	}

	target := uint64(head.Header.Message.Slot)
	if target > uint64(t.config.HeadOffset) {
		target -= uint64(t.config.HeadOffset)
	}

	return t.fetchHeaderBySlot(ctx, rpcClient, target)
}

func (t *Task) fetchHeaderBySlot(ctx context.Context, rpcClient *rpc.BeaconClient, target uint64) (*v1.BeaconBlockHeader, error) {
	var lastErr error

	for i := 0; i < t.config.MaxLookback; i++ {
		if target == 0 {
			break
		}

		hCtx, cancel := context.WithTimeout(ctx, t.config.RequestTimeout.Duration)
		raw, err := rpcClient.GetBlockHeaderBySlot(hCtx, phase0.Slot(target))

		cancel()

		if err == nil && raw != nil {
			return raw, nil
		}

		lastErr = err
		target--
	}

	if lastErr != nil {
		return nil, fmt.Errorf("could not resolve slot to a canonical block: %w", lastErr)
	}

	return nil, fmt.Errorf("could not resolve slot to a canonical block")
}

// parseRoot accepts either "0x..." or "..." hex and returns a 32-byte
// root, padding/truncating as needed.
func parseRoot(s string) (phase0.Root, error) {
	clean := strings.TrimPrefix(s, "0x")

	b, err := hex.DecodeString(clean)
	if err != nil {
		return phase0.Root{}, fmt.Errorf("invalid block root %q: %w", s, err)
	}

	if len(b) != 32 {
		return phase0.Root{}, fmt.Errorf("invalid block root length %d (want 32)", len(b))
	}

	var out phase0.Root
	copy(out[:], b)

	return out, nil
}
