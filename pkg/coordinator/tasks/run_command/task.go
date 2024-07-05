package runcommand

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_command"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs a shell command.",
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
