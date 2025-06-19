package getexecutionblock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
)

var (
	TaskName       = "get_execution_block"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Gets the latest execution block.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
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

func (t *Task) Execute(_ context.Context) error {
	t.logger.Info("Getting the latest execution block...")

	executionPool := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool()
	blocks := executionPool.GetBlockCache().GetCachedBlocks()

	if len(blocks) == 0 || blocks[0] == nil || blocks[0].GetBlock() == nil {
		t.logger.Error("No blocks found or the first block is nil")
		return errors.New("no blocks found or the first block is nil")
	}

	block := blocks[0].GetBlock()

	t.logger.Infof("Fetched block number %v", block.Number())
	t.logger.Infof("Fetched block hash %v", block.Hash().Hex())

	if headerData, err := vars.GeneralizeData(block.Header()); err != nil {
		t.logger.Warnf("Failed encoding block #%v header for 'header' output: %v", block.Number(), err)
	} else {
		t.ctx.Outputs.SetVar("header", headerData)
	}

	return nil
}
