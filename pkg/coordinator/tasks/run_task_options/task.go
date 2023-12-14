package runtaskoptions

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_task_options"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs task with configurable behaviour.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	task    types.Task
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Scheduler.GetLogger().WithField("task", TaskName),
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
	if err2 := config.Validate(); err2 != nil {
		return err2
	}

	// init child task
	taskOpts, err := t.ctx.Scheduler.ParseTaskOptions(config.Task)
	if err != nil {
		return fmt.Errorf("failed parsing child task config: %w", err)
	}

	taskVars := t.ctx.Vars
	if config.NewVariableScope {
		taskVars = taskVars.NewScope()
	}

	t.task, err = t.ctx.NewTask(taskOpts, taskVars)
	if err != nil {
		return fmt.Errorf("failed initializing child task: %w", err)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	err := t.ctx.Scheduler.ExecuteTask(ctx, t.task, func(ctx context.Context, cancelFn context.CancelFunc, _ types.Task) {
		t.watchTaskResult(ctx, cancelFn)
	})

	switch {
	case t.config.ExpectFailure:
		if err == nil {
			return fmt.Errorf("child task succeeded, but should have failed")
		}
	case t.config.IgnoreFailure:
		if err != nil {
			t.logger.Warnf("child task failed: %w", err)
		}
	default:
		if err != nil {
			return fmt.Errorf("child task failed: %w", err)
		}
	}

	return nil
}

func (t *Task) watchTaskResult(ctx context.Context, cancelFn context.CancelFunc) {
	currentResult := types.TaskResultNone

	for {
		updateChan := t.ctx.Scheduler.GetTaskResultUpdateChan(t.task, types.TaskResultNone)
		if updateChan != nil {
			select {
			case <-ctx.Done():
				return
			case <-updateChan:
			}
		}

		taskStatus := t.ctx.Scheduler.GetTaskStatus(t.task)
		if taskStatus.Result == currentResult {
			continue
		}

		currentResult = taskStatus.Result

		taskResult := currentResult
		if t.config.InvertResult {
			switch taskResult {
			case types.TaskResultNone:
				taskResult = types.TaskResultSuccess
			case types.TaskResultSuccess:
				taskResult = types.TaskResultNone
			case types.TaskResultFailure:
				if t.config.ExpectFailure || t.config.IgnoreFailure {
					taskResult = types.TaskResultSuccess
				}
			}
		}

		t.ctx.SetResult(taskResult)

		if t.config.ExitOnResult {
			cancelFn()
			return
		}
	}
}
