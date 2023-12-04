package awaitconsensusblockproposal

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/minccino/pkg/coordinator/types"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "await_consensus_block_proposal"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Await a consensus block proposal that meets specific criteria.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx         *types.TaskContext
	options     *types.TaskOptions
	config      Config
	logger      logrus.FieldLogger
	firstHeight map[uint16]uint64
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	config := DefaultConfig()
	if options.Config != nil {
		conf := &Config{}
		if err := options.Config.Unmarshal(&conf); err != nil {
			return nil, fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
		if err := mergo.Merge(&config, conf, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging task config for %v: %w", TaskName, err)
		}
	}
	return &Task{
		ctx:         ctx,
		options:     options,
		config:      config,
		logger:      ctx.Scheduler.GetLogger().WithField("task", TaskName),
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
	return t.options.Title
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

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetCoordinator().ClientPool().GetConsensusPool()
	blockSubscription := consensusPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	totalMatches := 0
	for {
		select {
		case block := <-blockSubscription.Channel():
			matches := t.checkBlock(ctx, block)
			if !matches {
				break
			}
			totalMatches++
			t.logger.Infof("matching block %v [0x%x]", block.Slot, block.Root)

			if totalMatches >= t.config.BlockCount {
				t.ctx.SetResult(types.TaskResultSuccess)
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) checkBlock(ctx context.Context, block *consensus.Block) bool {
	blockData := block.AwaitBlock(2*time.Second, ctx)
	if blockData == nil {
		t.logger.Warnf("could not fetch block data for block %v [0x%x]", block.Slot, block.Root)
		return false
	}

	// check graffiti
	if len(t.config.GraffitiPatterns) > 0 {
		graffiti, err := blockData.Graffiti()
		if err != nil {
			t.logger.Warnf("could not get graffiti for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		var matched bool
		for _, pattern := range t.config.GraffitiPatterns {
			matched, err = regexp.MatchString(pattern, string(graffiti[:]))
			if matched || err != nil {
				break
			}
		}
		if err != nil {
			t.logger.Warnf("could not check graffiti for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if !matched {
			t.logger.Debugf("check failed for block %v [0x%x]: unmatched graffiti", block.Slot, block.Root)
			return false
		}
	}

	return true
}
