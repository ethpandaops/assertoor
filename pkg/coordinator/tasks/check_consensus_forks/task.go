package checkconsensusforks

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_forks"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check for consensus layer forks.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	startEpoch uint64
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
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
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	blockSubscription := consensusPool.GetBlockCache().SubscribeBlockEvent(10)

	defer blockSubscription.Unsubscribe()

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return fmt.Errorf("failed fetching wallclock: %w", err)
	}

	t.startEpoch = currentEpoch.Number()

	for {
		select {
		case <-blockSubscription.Channel():
			t.ctx.SetResult(t.runCheck())
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) runCheck() types.TaskResult {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	headForks := consensusPool.GetHeadForks(int64(t.config.MaxForkDistance))
	if len(headForks)-1 > int(t.config.MaxForkCount) {
		t.logger.Warnf("check failed: too many forks. (have: %v, want <= %v)", len(headForks)-1, t.config.MaxForkCount)

		for idx, fork := range headForks {
			t.logger.Infof("Fork #%v: %v [0x%x] (%v clients)", idx, fork.Slot, fork.Root, len(fork.AllClients))
		}

		return types.TaskResultFailure
	}

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

	return types.TaskResultSuccess
}
