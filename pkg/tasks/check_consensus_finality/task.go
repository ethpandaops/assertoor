package checkconsensusfinality

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_finality"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check finality status for consensus chain.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "finalizedEpoch",
				Type:        "uint64",
				Description: "The latest finalized epoch number.",
			},
			{
				Name:        "finalizedRoot",
				Type:        "string",
				Description: "The root hash of the finalized checkpoint.",
			},
			{
				Name:        "unfinalizedEpochs",
				Type:        "uint64",
				Description: "Number of epochs since the last finalized checkpoint.",
			},
		},
		NewTask: NewTask,
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
	return &Task{
		ctx:         ctx,
		options:     options,
		logger:      ctx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
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

	wallclockSubscription := consensusPool.GetBlockCache().SubscribeWallclockEpochEvent(10)
	defer wallclockSubscription.Unsubscribe()

	checkpointSubscription := consensusPool.GetBlockCache().SubscribeFinalizedEvent(10)
	defer checkpointSubscription.Unsubscribe()

	checkCount := 0

	for {
		checkCount++
		checkResult, status := t.runFinalityCheck()

		switch {
		case checkResult:
			t.ctx.SetResult(types.TaskResultSuccess)
			t.ctx.ReportProgress(100, "Finality check passed")

			if !t.config.ContinueOnPass {
				return nil
			}
		case t.config.FailOnCheckMiss:
			t.ctx.SetResult(types.TaskResultFailure)
			t.ctx.ReportProgress(0, fmt.Sprintf("Finality check failed: %s (attempt %d)", status, checkCount))

			return fmt.Errorf("finality check failed: %s", status)
		default:
			t.ctx.SetResult(types.TaskResultNone)
			t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for finality, %s (attempt %d)", status, checkCount))
		}

		select {
		case epoch := <-wallclockSubscription.Channel():
			t.logger.Infof("wallclock epoch %v", epoch.Number())
		case checkpoint := <-checkpointSubscription.Channel():
			t.logger.Infof("new checkpoint %v", checkpoint.Epoch)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) runFinalityCheck() (passed bool, status string) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		t.logger.Warnf("failed fetching wallclock: %v", err.Error())
		return false, fmt.Sprintf("failed fetching wallclock: %v", err)
	}

	finalizedEpoch, finalizedRoot := consensusPool.GetBlockCache().GetFinalizedCheckpoint()
	unfinalizedEpochs := currentEpoch.Number() - uint64(finalizedEpoch)

	t.ctx.Outputs.SetVar("finalizedEpoch", finalizedEpoch)
	t.ctx.Outputs.SetVar("finalizedRoot", finalizedRoot.String())
	t.ctx.Outputs.SetVar("unfinalizedEpochs", unfinalizedEpochs)

	if t.config.MinUnfinalizedEpochs > 0 && unfinalizedEpochs < t.config.MinUnfinalizedEpochs {
		msg := fmt.Sprintf("%v unfinalized epochs < %v (finalized epoch %v, current epoch %v)", unfinalizedEpochs, t.config.MinUnfinalizedEpochs, finalizedEpoch, currentEpoch.Number())
		t.logger.Infof("check failed: %v", msg)

		return false, msg
	}

	if t.config.MaxUnfinalizedEpochs > 0 && unfinalizedEpochs > t.config.MaxUnfinalizedEpochs {
		msg := fmt.Sprintf("%v unfinalized epochs > %v (finalized epoch %v, current epoch %v)", unfinalizedEpochs, t.config.MaxUnfinalizedEpochs, finalizedEpoch, currentEpoch.Number())
		t.logger.Infof("check failed: %v", msg)

		return false, msg
	}

	if t.config.MinFinalizedEpochs > 0 && uint64(finalizedEpoch) < t.config.MinFinalizedEpochs {
		msg := fmt.Sprintf("finalized epoch %v < %v (current epoch %v)", finalizedEpoch, t.config.MinFinalizedEpochs, currentEpoch.Number())
		t.logger.Infof("check failed: %v", msg)

		return false, msg
	}

	return true, ""
}
