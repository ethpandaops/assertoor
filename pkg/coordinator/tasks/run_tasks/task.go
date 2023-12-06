package runtasks

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/imdario/mergo"
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
	tasks   []types.Task
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	config := DefaultConfig()

	if options.Config != nil {
		conf := &Config{}
		if err := options.Config.Unmarshal(&conf); err != nil {
			return nil, fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}

		if err := mergo.Merge(&config, conf, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging task config for %v: %w", TaskName, err)
		}
	}

	childTasks := []types.Task{}

	for i := range config.Tasks {
		taskOpts, err := ctx.Scheduler.ParseTaskOptions(&config.Tasks[i])
		if err != nil {
			return nil, fmt.Errorf("failed parsing child task config #%v : %w", i, err)
		}

		task, err := ctx.NewTask(taskOpts)
		if err != nil {
			return nil, fmt.Errorf("failed initializing child task #%v : %w", i, err)
		}

		childTasks = append(childTasks, task)
	}

	return &Task{
		ctx:     ctx,
		options: options,
		config:  config,
		logger:  ctx.Scheduler.GetLogger().WithField("task", TaskName),
		tasks:   childTasks,
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.options.Title
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

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	for i, task := range t.tasks {
		err := t.ctx.Scheduler.ExecuteTask(ctx, task, t.ctx.Scheduler.WatchTaskPass)
		if err != nil {
			return fmt.Errorf("child task #%v failed: %w", i, err)
		}
	}

	return nil
}
