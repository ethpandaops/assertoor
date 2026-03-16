package checkconsensusbuildersstatus

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_builder_status"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check builder status on consensus chain by loading the full beacon state.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "builder",
				Type:        "object",
				Description: "The builder information object.",
			},
			{
				Name:        "builderIndex",
				Type:        "number",
				Description: "The builder's index in the builder list.",
			},
			{
				Name:        "pubkey",
				Type:        "string",
				Description: "The builder's public key.",
			},
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

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if valerr := config.Validate(); valerr != nil {
		return valerr
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	wallclockEpochSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer wallclockEpochSubscription.Unsubscribe()

	checkCount := 0

	checkCount++

	if done, err := t.processCheck(ctx, checkCount); done {
		return err
	}

	for {
		select {
		case <-wallclockEpochSubscription.Channel():
			checkCount++

			if done, err := t.processCheck(ctx, checkCount); done {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) processCheck(ctx context.Context, checkCount int) (bool, error) {
	checkResult := t.runBuilderStatusCheck(ctx)

	_, epoch, _ := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().GetWallclock().Now()
	t.logger.Infof("epoch %v check result: %v", epoch.Number(), checkResult)

	switch {
	case checkResult:
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, fmt.Sprintf("Builder status check passed at epoch %d", epoch.Number()))

		if !t.config.ContinueOnPass {
			return true, nil
		}

		return false, nil
	case t.config.FailOnCheckMiss:
		t.ctx.SetResult(types.TaskResultFailure)
		t.ctx.ReportProgress(0, fmt.Sprintf("Builder status check failed at epoch %d", epoch.Number()))

		return true, fmt.Errorf("builder status check failed at epoch %d", epoch.Number())
	default:
		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for builder status... (attempt %d)", checkCount))

		return false, nil
	}
}

func (t *Task) runBuilderStatusCheck(_ context.Context) bool {
	builderSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBuilderSet()
	if len(builderSet) == 0 {
		t.logger.Infof("check failed: no builders in builder set")
		return false
	}

	pubkey := []byte{}
	if t.config.BuilderPubKey != "" {
		pubkey = common.FromHex(t.config.BuilderPubKey)
	}

	for _, info := range builderSet {
		builder := info.Builder

		if t.config.BuilderIndex != nil && uint64(info.Index) != *t.config.BuilderIndex {
			continue
		}

		if t.config.BuilderPubKey != "" && !bytes.Equal(pubkey, builder.PublicKey[:]) {
			continue
		}

		// Found a matching builder
		t.logger.Infof("builder found: index %v, pubkey 0x%x, balance %v, deposit_epoch %v, withdrawable_epoch %v",
			info.Index, builder.PublicKey[:], builder.Balance, builder.DepositEpoch, builder.WithdrawableEpoch)

		if body, err := vars.GeneralizeData(builder); err == nil {
			t.ctx.Outputs.SetVar("builder", body)
		} else {
			t.logger.Warnf("failed encoding builder info: %v", err)
		}

		t.ctx.Outputs.SetVar("builderIndex", uint64(info.Index))
		t.ctx.Outputs.SetVar("pubkey", fmt.Sprintf("0x%x", builder.PublicKey[:]))

		// FAR_FUTURE_EPOCH sentinel value
		farFutureEpoch := uint64(0xFFFFFFFFFFFFFFFF)

		// is_active_builder: deposit_epoch < finalized_epoch AND withdrawable_epoch == FAR_FUTURE
		if t.config.ExpectActive {
			if uint64(builder.WithdrawableEpoch) != farFutureEpoch {
				t.logger.Infof("check failed: expected active builder but withdrawable_epoch is %v", builder.WithdrawableEpoch)
				continue
			}

			finalizedEpoch, _ := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().GetFinalizedCheckpoint()
			if builder.DepositEpoch >= finalizedEpoch {
				t.logger.Infof("check failed: builder deposit not yet finalized (deposit_epoch: %v, finalized_epoch: %v)", builder.DepositEpoch, finalizedEpoch)
				continue
			}
		}

		if t.config.ExpectExiting && uint64(builder.WithdrawableEpoch) == farFutureEpoch {
			t.logger.Infof("check failed: expected exiting builder but withdrawable_epoch is FAR_FUTURE")
			continue
		}

		if t.config.MinBuilderBalance > 0 && uint64(builder.Balance) < t.config.MinBuilderBalance {
			t.logger.Infof("check failed: builder balance below minimum: %v", builder.Balance)
			continue
		}

		if t.config.MaxBuilderBalance != nil && uint64(builder.Balance) > *t.config.MaxBuilderBalance {
			t.logger.Infof("check failed: builder balance above maximum: %v", builder.Balance)
			continue
		}

		return true
	}

	t.logger.Infof("check failed: no matching builder found")

	return false
}
