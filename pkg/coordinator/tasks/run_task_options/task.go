package runtaskoptions

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
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

		// execute task
		taskErr = t.ctx.Scheduler.ExecuteTask(ctx, t.task, func(ctx context.Context, cancelFn context.CancelFunc, _ types.TaskIndex) {
			t.watchTaskResult(ctx, cancelFn)
		})

		switch {
		case t.config.RetryOnFailure && retryCount < t.config.MaxRetryCount:
			if taskErr != nil {
				retryCount++

				t.logger.Warnf("child task failed: %w (retrying)", taskErr)

				continue
			}
		case t.config.ExpectFailure:
			if taskErr == nil {
				t.ctx.SetResult(types.TaskResultFailure)
				return fmt.Errorf("child task succeeded, but should have failed")
			} else {
				t.ctx.SetResult(types.TaskResultSuccess)
			}
		case t.config.IgnoreFailure:
			if taskErr != nil {
				t.logger.Warnf("child task failed: %w", taskErr)
			}

			t.ctx.SetResult(types.TaskResultSuccess)
		default:
			if taskErr != nil {
				t.ctx.SetResult(types.TaskResultFailure)
				return fmt.Errorf("child task failed: %w", taskErr)
			}
		}

		break
	}

	return taskErr
}

func (t *Task) watchTaskResult(ctx context.Context, cancelFn context.CancelFunc) {
	taskState := t.ctx.Scheduler.GetTaskState(t.task)
	currentResult := types.TaskResultNone

	for {
		updateChan := taskState.GetTaskResultUpdateChan(currentResult)
		if updateChan != nil {
			select {
			case <-ctx.Done():
				return
			case <-updateChan:
			}
		}

		taskStatus := taskState.GetTaskStatus()
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

		if t.config.PropagateResult {
			t.ctx.SetResult(taskResult)
		}

		if t.config.ExitOnResult {
			cancelFn()
			return
		}
	}
}
