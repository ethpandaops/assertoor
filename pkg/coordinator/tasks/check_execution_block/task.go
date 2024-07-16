package checkexecutionblock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_execution_block"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks the latest execution block.",
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

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) Execute(ctx context.Context) error {
	t.logger.Info("Checking the latest execution block...")

	executionPool := t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool()
	blocks := executionPool.GetBlockCache().GetCachedBlocks()

	if len(blocks) == 0 || blocks[0] == nil || blocks[0].GetBlock() == nil {
		t.logger.Error("No blocks found or the first block is nil")
		return errors.New("no blocks found or the first block is nil")
	}

	block := blocks[0].GetBlock()

	t.logger.Infof("Fetched block number %v", block.Number())
	t.logger.Infof("Fetched block hash %v", block.Hash().Hex())
	jsonBytes, err := json.Marshal(block.Header())
	if err != nil {
		t.logger.Errorf("Error marshaling block to JSON: %v", err)
		return err
	}
	t.ctx.Vars.SetVar(t.config.BlockHeaderResultVar, string(jsonBytes))

	return nil
}
