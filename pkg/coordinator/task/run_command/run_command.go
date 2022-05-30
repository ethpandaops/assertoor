package runcommand

import (
	"context"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

type Task struct {
	config Config
	log    logrus.FieldLogger
}

const (
	Name        = "run_command"
	Description = "Runs a shell command."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, config Config) *Task {
	return &Task{
		config: config,
		log:    log.WithField("task", Name),
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *Task) PollingInterval() time.Duration {
	return time.Second * 5
}

func (t *Task) Start(ctx context.Context) error {
	if len(t.config.Command) == 0 {
		return nil
	}

	com := t.config.Command[0]
	args := []string{}

	if len(t.config.Command) > 1 {
		args = t.config.Command[1:]
	}

	t.Logger().WithField("cmd", com).WithField("args", args).WithField("allowed_to_failed", t.config.AllowedToFail).Info("running command")

	command := exec.CommandContext(ctx, com, args...)

	stdOut, err := command.CombinedOutput()
	if err != nil {
		t.Logger().WithField("stdout", string(stdOut)).WithError(err).Error("failed to run command")

		if t.config.AllowedToFail {
			return nil
		}

		return err
	}

	t.Logger().WithField("stdout", string(stdOut)).Info("command run successfully")

	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
