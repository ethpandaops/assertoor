package runtaskoptions

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_task_options"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs task with configurable behaviour.",
		Category:    "flow-control",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	task    types.TaskIndex
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
	if err2 := config.Validate(); err2 != nil {
		return err2
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	var taskErr error

	retryCount := uint(0)

	t.ctx.ReportProgress(0, "Running child task...")

	for {
		// init child task
		taskOpts, err := t.ctx.Scheduler.ParseTaskOptions(t.config.Task)
		if err != nil {
			return fmt.Errorf("failed parsing child task config: %w", err)
		}

		taskVars := t.ctx.Vars
		if t.config.NewVariableScope {
			taskVars = taskVars.NewScope()
			taskVars.SetVar("scopeOwner", uint64(t.ctx.Index))
			t.ctx.Outputs.SetSubScope("childScope", vars.NewScopeFilter(taskVars))
		}

		t.task, err = t.ctx.NewTask(taskOpts, taskVars)
		if err != nil {
			return fmt.Errorf("failed initializing child task: %w", err)
		}

		// execute task (tasks self-complete now)
		taskErr = t.ctx.Scheduler.ExecuteTask(ctx, t.task, nil)

		// Handle retry logic
		if t.config.RetryOnFailure && taskErr != nil && retryCount < t.config.MaxRetryCount {
			retryCount++

			t.logger.Warnf("child task failed: %v (retrying)", taskErr)
			t.ctx.ReportProgress(0, fmt.Sprintf("Retrying child task (attempt %d/%d)...", retryCount+1, t.config.MaxRetryCount+1))

			continue
		}

		break
	}

	t.ctx.ReportProgress(100, "Task completed")

	// Apply result transformation
	if t.config.IgnoreResult {
		return nil
	}

	// ExpectFailure is an alias for InvertResult
	if t.config.ExpectFailure || t.config.InvertResult {
		if taskErr != nil {
			return nil
		}

		return fmt.Errorf("child task succeeded, but failure was expected")
	}

	return taskErr
}
