package runshell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_shell"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs commands in a shell.",
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
		logger:  ctx.Scheduler.GetLogger().WithField("task", TaskName),
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
	cmdLogger := t.logger.WithField("shell", t.config.Shell)
	cmdLogger.Info("running command")

	//nolint:gosec // ignore
	command := exec.CommandContext(ctx, t.config.Shell)

	stdin, err := command.StdinPipe()
	if err != nil {
		cmdLogger.Errorf("failed getting stdin pipe")
		return err
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}

	stdoutChan := t.readOutputStream(stdout)
	defer close(stdoutChan)

	stderrChan := t.readOutputStream(stderr)
	defer close(stderrChan)

	// start shell
	err = command.Start()
	if err != nil {
		cmdLogger.Errorf("failed starting shell")
		return err
	}

	// write command to stdin
	_, err = io.WriteString(stdin, t.config.Command+"\n")
	if err != nil {
		cmdLogger.Errorf("failed writing command to stdin pipe")
		return err
	}

	stdin.Close()

	// wait for process
	var execErr error

	waitChan := make(chan bool)
	go func() {
		defer close(waitChan)

		execErr = command.Wait()
	}()

	// wait for output handler
cmdloop:
	for {
		select {
		case line := <-stdoutChan:
			cmdLogger.Infof("OUT: %v", line)
		case line := <-stderrChan:
			cmdLogger.Warnf("ERR: %v", line)
		case <-waitChan:
			break cmdloop
		}
	}

	// await completion
	if execErr != nil {
		cmdLogger.Errorf("failed command execution")
		return err
	}

	cmdLogger.Info("command run successfully")

	return nil
}

func (t *Task) readOutputStream(pipe io.ReadCloser) chan string {
	resChan := make(chan string)

	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			if scanner.Err() != nil {
				return
			}

			resChan <- scanner.Text()
		}
	}()

	return resChan
}
