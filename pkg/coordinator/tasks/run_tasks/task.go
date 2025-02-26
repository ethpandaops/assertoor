package runtasks

import (
	"context"
	"fmt"
	"time"

	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_tasks"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Run tasks sequentially.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	tasks   []types.TaskIndex
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

	// init child tasks
	childTasks := []types.TaskIndex{}

	var taskVars types.Variables

	if t.config.NewVariableScope {
		taskVars = t.ctx.Vars.NewScope()
		taskVars.SetVar("scopeOwner", uint64(t.ctx.Index))
		t.ctx.Outputs.SetSubScope("childScope", taskVars)
	}

	for i := range config.Tasks {
		taskOpts, err := t.ctx.Scheduler.ParseTaskOptions(&config.Tasks[i])
		if err != nil {
			return fmt.Errorf("failed parsing child task config #%v : %w", i+1, err)
		}

		task, err := t.ctx.NewTask(taskOpts, taskVars)
		if err != nil {
			return fmt.Errorf("failed initializing child task #%v : %w", i+1, err)
		}

		childTasks = append(childTasks, task)
	}

	t.config = config
	t.tasks = childTasks

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	for i, task := range t.tasks {
		err := t.ctx.Scheduler.ExecuteTask(ctx, task, func(ctx context.Context, cancelFn context.CancelFunc, task types.TaskIndex) {
			if t.config.StopChildOnResult {
				t.ctx.Scheduler.WatchTaskPass(ctx, cancelFn, task)
			}
		})

		switch {
		case t.config.ExpectFailure:
			if err == nil {
				return fmt.Errorf("child task #%v succeeded, but should have failed", i+1)
			}
		case t.config.ContinueOnFailure:
			if err != nil {
				t.logger.Warnf("child task #%v failed: %w", i+1, err)
			}
		default:
			if err != nil {
				return fmt.Errorf("child task #%v failed: %w", i+1, err)
			}
		}
	}

	return nil
}
