package checkconsensusreorgs

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_reorgs"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check for consensus layer reorgs.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx              *types.TaskContext
	options          *types.TaskOptions
	config           Config
	logger           logrus.FieldLogger
	startEpoch       uint64
	totalReorgs      uint64
	maxReorgDistance uint64
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

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return fmt.Errorf("failed fetching wallclock: %w", err)
	}

	t.startEpoch = currentEpoch.Number()

	var lastBlock *consensus.Block

	for {
		select {
		case block := <-blockSubscription.Channel():
			blockHeader := block.AwaitHeader(ctx, 500*time.Millisecond)
			if blockHeader == nil {
				break
			}

			if lastBlock != nil && !bytes.Equal(blockHeader.Message.ParentRoot[:], lastBlock.Root[:]) {
				// chain reorg
				t.processChainReorg(lastBlock, block)

				if t.config.MaxReorgDistance > 0 && t.maxReorgDistance > t.config.MaxReorgDistance {
					t.ctx.SetResult(types.TaskResultFailure)
					t.logger.Infof("task failed: max reorg distance (%v) exceeded, reorg distance around slot %v: %v", t.config.MaxReorgDistance, block.Slot, t.maxReorgDistance)

					return fmt.Errorf("max reorg distance (%v) exceeded:  %v -> %v (%v)", t.config.MaxReorgDistance, lastBlock.Root.String(), block.Root.String(), t.maxReorgDistance)
				}
			}

			t.ctx.SetResult(t.runCheck())

			lastBlock = block
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) runCheck() types.TaskResult {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		t.logger.Warnf("check missed: could not get current epoch from wall clock")
		return types.TaskResultNone
	}

	epochCount := currentEpoch.Number() - t.startEpoch

	if t.config.MinCheckEpochCount > 0 && epochCount < t.config.MinCheckEpochCount {
		t.logger.Warnf("Check missed: checked %v epochs, but need >= %v", epochCount, t.config.MinCheckEpochCount)
		return types.TaskResultNone
	}

	if t.config.MaxReorgsPerEpoch > 0 && epochCount > 0 && float64(t.totalReorgs)/float64(epochCount) > t.config.MaxReorgsPerEpoch {
		t.logger.Warnf("check failed: max reorgs per epoch exceeded. (have: %v, want <= %v)", float64(t.totalReorgs)/float64(epochCount), t.config.MaxReorgsPerEpoch)
		return types.TaskResultFailure
	}

	return types.TaskResultSuccess
}

func (t *Task) processChainReorg(oldHead, newHead *consensus.Block) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	parentHead := oldHead
	newHeadDistance := uint64(0)
	oldHeadDistance := uint64(0)
	parentUnknown := false

	for {
		parentRoot := parentHead.GetParentRoot()
		if parentRoot == nil {
			parentUnknown = true
			break
		}

		parentHead = consensusPool.GetBlockCache().GetCachedBlockByRoot(*parentRoot)
		if parentHead == nil {
			parentUnknown = true
			break
		}

		oldHeadDistance++

		linked, newHeadDist := consensusPool.GetBlockCache().GetBlockDistance(parentHead.Root, newHead.Root)

		if linked {
			newHeadDistance = newHeadDist
			break
		}
	}

	if parentUnknown {
		t.logger.Warnf("cannot get reorg distance: unknown base parent")
	}

	reorgDistance := oldHeadDistance + newHeadDistance
	t.logger.Infof("chain reorg (distance: %v, old head: %v, new head: %v)", reorgDistance, oldHead.Root.String(), newHead.Root.String())

	if reorgDistance > t.maxReorgDistance {
		t.maxReorgDistance = reorgDistance
	}

	t.totalReorgs++
}
