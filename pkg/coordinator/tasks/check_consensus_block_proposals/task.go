package checkconsensusblockproposals

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
	TaskName       = "check_consensus_block_proposals"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check for consensus block proposals that meet specific criteria.",
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

	// check deposit count
	if t.config.MinDepositCount > 0 {
		deposits, err := blockData.Deposits()
		if err != nil {
			t.logger.Warnf("could not get deposits for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if len(deposits) < t.config.MinDepositCount {
			t.logger.Debugf("check failed for block %v [0x%x]: not enough deposits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinDepositCount, len(deposits))
			return false
		}
	}

	// check exit count
	if t.config.MinExitCount > 0 {
		exits, err := blockData.VoluntaryExits()
		if err != nil {
			t.logger.Warnf("could not get voluntary exits for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if len(exits) < t.config.MinExitCount {
			t.logger.Debugf("check failed for block %v [0x%x]: not enough exits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinExitCount, len(exits))
			return false
		}
	}

	// check exit count
	if t.config.MinSlashingCount > 0 {
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
			t.logger.Debugf("check failed for block %v [0x%x]: not enough exits (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinSlashingCount, slashingCount)
			return false
		}
	}

	// check bls change count
	if t.config.MinBlsChangeCount > 0 {
		blsChanges, err := blockData.BLSToExecutionChanges()
		if err != nil {
			t.logger.Warnf("could not get bls to execution changes for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if len(blsChanges) < t.config.MinBlsChangeCount {
			t.logger.Debugf("check failed for block %v [0x%x]: not enough bls changes (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinBlsChangeCount, len(blsChanges))
			return false
		}
	}

	// check withdrawal count
	if t.config.MinWithdrawalCount > 0 {
		withdrawals, err := blockData.Withdrawals()
		if err != nil {
			t.logger.Warnf("could not get withdrawals for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if len(withdrawals) < t.config.MinWithdrawalCount {
			t.logger.Debugf("check failed for block %v [0x%x]: not enough withdrawals (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinWithdrawalCount, len(withdrawals))
			return false
		}
	}

	// check transaction count
	if t.config.MinTransactionCount > 0 {
		transactions, err := blockData.ExecutionTransactions()
		if err != nil {
			t.logger.Warnf("could not get transactions for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if len(transactions) < t.config.MinTransactionCount {
			t.logger.Debugf("check failed for block %v [0x%x]: not enough transactions (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinTransactionCount, len(transactions))
			return false
		}
	}

	// check blob count
	if t.config.MinBlobCount > 0 {
		blobs, err := blockData.BlobKzgCommitments()
		if err != nil {
			t.logger.Warnf("could not get blobs for block %v [0x%x]: %v", block.Slot, block.Root, err)
			return false
		}
		if len(blobs) < t.config.MinBlobCount {
			t.logger.Debugf("check failed for block %v [0x%x]: not enough blobs (want: >= %v, have: %v)", block.Slot, block.Root, t.config.MinBlobCount, len(blobs))
			return false
		}
	}

	return true
}