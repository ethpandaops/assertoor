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
}

var _ Runnable = (*RunCommand)(nil)

const (
	NameRunCommand = "run_command"
)

func NewRunCommand(ctx context.Context, bundle *Bundle, cmd ...string) *RunCommand {
	bundle.log = bundle.log.WithField("task", NameRunCommand)

	return &RunCommand{
		bundle: bundle,
		cmd:    cmd,
	}
}

func (b *RunCommand) Name() string {
	return NameRunCommand
}

func (b *RunCommand) PollingInterval() time.Duration {
	return time.Second * 5
}

func (b *RunCommand) Start(ctx context.Context) error {
	b.Logger().WithField("cmd", b.cmd).Info("running command")

	command := exec.CommandContext(ctx, b.cmd[0], b.cmd[0:]...) //nolint:gosec // We know what we're doing here

	stdOut, err := command.CombinedOutput()
	if err != nil {
		b.Logger().WithField("stdout", string(stdOut)).WithError(err).Error("failed to run command")
		return err
	}

	b.Logger().WithField("stdout", string(stdOut)).Info("command run successfully")

	return nil
}

func (b *RunCommand) Logger() logrus.FieldLogger {
	return b.bundle.Logger()
}

func (b *RunCommand) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
