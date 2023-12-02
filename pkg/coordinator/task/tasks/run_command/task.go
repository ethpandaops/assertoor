package runcommand

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/task/types"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_command"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs a shell command.",
		Config:      Config{},
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
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.options.Title
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
	if taskStatus.IsRunning {
		return types.TaskResultNone, nil
	}
	return types.TaskResultSuccess, nil
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}
	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if len(t.config.Command) == 0 {
		return nil
	}

	com := t.config.Command[0]
	args := []string{}

	if len(t.config.Command) > 1 {
		args = t.config.Command[1:]
	}

	t.logger.WithField("cmd", com).WithField("args", args).WithField("allowed_to_failed", t.config.AllowedToFail).Info("running command")

	command := exec.CommandContext(ctx, com, args...)

	stdOut, err := command.CombinedOutput()
	if err != nil {
		t.logger.WithField("stdout", string(stdOut)).WithError(err).Error("failed to run command")

		if t.config.AllowedToFail {
			return nil
		}

		return err
	}

	t.logger.WithField("stdout", string(stdOut)).Info("command run successfully")

	return nil
}

func (t *Task) Cleanup(ctx context.Context) error {
	return nil
}
