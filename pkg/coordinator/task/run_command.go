package task

import (
	"context"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

type RunCommand struct {
	bundle *Bundle
	cmd    []string
	log    logrus.FieldLogger
}

var _ Runnable = (*RunCommand)(nil)

const (
	NameRunCommand = "run_command"
)

func NewRunCommand(ctx context.Context, bundle *Bundle, cmd ...string) *RunCommand {
	return &RunCommand{
		bundle: bundle,
		cmd:    cmd,
		log:    bundle.log.WithField("task", NameRunCommand),
	}
}

func (b *RunCommand) Name() string {
	return NameRunCommand
}

func (b *RunCommand) PollingInterval() time.Duration {
	return time.Second * 5
}

func (b *RunCommand) Start(ctx context.Context) error {
	com := b.cmd[0]
	args := []string{}

	if len(b.cmd) > 1 {
		args = b.cmd[1:]
	}

	b.Logger().WithField("cmd", com).WithField("args", args).Info("running command")

	command := exec.CommandContext(ctx, com, args...)

	stdOut, err := command.CombinedOutput()
	if err != nil {
		b.Logger().WithField("stdout", string(stdOut)).WithError(err).Error("failed to run command")
		return err
	}

	b.Logger().WithField("stdout", string(stdOut)).Info("command run successfully")

	return nil
}

func (b *RunCommand) Logger() logrus.FieldLogger {
	return b.log
}

func (b *RunCommand) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
