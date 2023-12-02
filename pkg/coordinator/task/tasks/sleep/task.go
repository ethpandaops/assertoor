package sleep

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/task/types"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "sleep"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Sleeps for a specified duration.",
		Config:      Config{},
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	result  types.TaskResult
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
	return &Task{
		ctx:     ctx,
		options: options,
		config:  config,
		logger:  ctx.Logger.WithField("task", TaskName),
	}, nil
}

func (t *Task) Name() string {
	return TaskDescriptor.Name
}

func (t *Task) Title() string {
	return t.options.Title
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) PollingInterval() time.Duration {
	return 0
}

func (t *Task) GetResult() (types.TaskResult, error) {
	taskStatus := t.ctx.Scheduler.GetTaskStatus(t)
	if taskStatus.Error != nil {
		return types.TaskResultFailure, taskStatus.Error
	}
	return t.result, nil
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	time.Sleep(t.config.Duration.Duration)
	return nil
}

func (t *Task) Cleanup(ctx context.Context) error {
	return nil
}
