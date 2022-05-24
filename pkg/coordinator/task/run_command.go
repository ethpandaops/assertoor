package task

import (
	"context"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

type RunCommandConfig struct {
	Command []string `yaml:"command"`
}

type RunCommand struct {
	bundle *Bundle
	config RunCommandConfig
	log    logrus.FieldLogger
}

var _ Runnable = (*RunCommand)(nil)

const (
	NameRunCommand = "run_command"
)

func NewRunCommand(ctx context.Context, bundle *Bundle, config RunCommandConfig) *RunCommand {
	return &RunCommand{
		bundle: bundle,
		config: config,
		log:    bundle.log.WithField("task", NameRunCommand),
	}
}

func DefaultRunCommandConfig() RunCommandConfig {
	return RunCommandConfig{
		Command: []string{},
	}
}

func (b *RunCommand) Name() string {
	return NameRunCommand
}

func (b *RunCommand) PollingInterval() time.Duration {
	return time.Second * 5
}

func (b *RunCommand) Start(ctx context.Context) error {
	if len(b.config.Command) == 0 {
		return nil
	}

	com := b.config.Command[0]
	args := []string{}

	if len(b.config.Command) > 1 {
		args = b.config.Command[1:]
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
