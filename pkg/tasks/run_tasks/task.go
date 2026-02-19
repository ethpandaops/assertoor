package runtasks

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_tasks"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Run tasks sequentially.",
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
	totalTasks := len(t.tasks)

	var taskErr error

	for i, task := range t.tasks {
		err := t.ctx.Scheduler.ExecuteTask(ctx, task, nil)
		if err != nil {
			if t.config.ContinueOnFailure {
				t.logger.Warnf("child task #%v failed: %v", i+1, err)
			} else {
				taskErr = fmt.Errorf("child task #%v failed: %w", i+1, err)
				break
			}
		}

		// Report progress after each task completes
		completedTasks := i + 1
		progress := float64(completedTasks) / float64(totalTasks) * 100
		t.ctx.ReportProgress(progress, fmt.Sprintf("Task %d/%d completed", completedTasks, totalTasks))
	}

	// Apply result transformation
	if t.config.IgnoreResult {
		return nil
	}

	if t.config.InvertResult {
		if taskErr != nil {
			return nil
		}

		return fmt.Errorf("all tasks succeeded, but failure was expected")
	}

	return taskErr
}
