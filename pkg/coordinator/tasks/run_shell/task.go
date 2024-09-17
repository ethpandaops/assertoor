package runshell

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
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
	cmdLogger := t.logger.WithField("shell", t.config.Shell)
	cmdLogger.Info("running command")

	//nolint:gosec // ignore
	command := exec.CommandContext(ctx, t.config.Shell, t.config.ShellArgs...)

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

	stdoutChan, stdoutCloseChan := t.readOutputStream(stdout, cmdLogger.WithField("stream", "stdout"))
	defer close(stdoutChan)

	stderrChan, stderrCloseChan := t.readOutputStream(stderr, cmdLogger.WithField("stream", "stderr"))
	defer close(stderrChan)

	// add env vars
	for envName, varName := range t.config.EnvVars {
		varValue, varFound, err2 := t.ctx.Vars.ResolveQuery(varName)
		if err2 != nil {
			cmdLogger.Errorf("failed parsing var query for env variable %v: %v", envName, err2)
			return err2
		}

		if !varFound {
			continue
		}

		varJSON, err3 := json.Marshal(varValue)
		if err3 != nil {
			cmdLogger.Errorf("failed encoding env variable %v: %v", envName, err3)
			return err3
		}

		command.Env = append(command.Env, fmt.Sprintf("%v=%v", envName, string(varJSON)))
	}

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

	// wait for process & output streams
	var execErr error

	waitChan := make(chan bool)
	go func() {
		defer close(waitChan)

		<-stdoutCloseChan
		<-stderrCloseChan

		execErr = command.Wait()
	}()

	// wait for output handler
cmdloop:
	for {
		select {
		case line := <-stdoutChan:
			if !t.parseOutputVars(line) {
				cmdLogger.Infof("OUT: %v", line)
			}
		case line := <-stderrChan:
			cmdLogger.Warnf("ERR: %v", line)
		case <-waitChan:
			break cmdloop
		}
	}

	// await completion
	if execErr != nil {
		cmdLogger.Errorf("failed command execution")
		return execErr
	}

	cmdLogger.Info("command run successfully")

	return nil
}

func (t *Task) readOutputStream(pipe io.ReadCloser, logger logrus.FieldLogger) (readChan chan string, closeChan chan bool) {
	readChan = make(chan string)
	closeChan = make(chan bool)

	go func() {
		var err error

		defer close(closeChan)

		reader := bufio.NewReader(pipe)

		for err == nil {
			isPrefix := true
			ln := []byte{}

			for isPrefix && err == nil {
				var line []byte

				line, isPrefix, err = reader.ReadLine()
				if err != nil {
					if err == io.EOF {
						break
					}

					logger.Errorf("error reading stream: %v", err)

					break
				}

				ln = append(ln, line...)
			}

			if len(ln) > 0 {
				readChan <- string(ln)
			}
		}
	}()

	return readChan, closeChan
}

var outputVarPattern = regexp.MustCompile(`^::set-var +([^ ]+) +(.*)$`)
var outputJSONPattern = regexp.MustCompile(`^::set-json +([^ ]+) +(.*)$`)
var outputOutPattern = regexp.MustCompile(`^::set-out(put)?(-json)? +([^ ]+) +(.*)$`)

func (t *Task) parseOutputVars(line string) bool {
	match := outputVarPattern.FindStringSubmatch(line)
	if match != nil {
		t.ctx.Vars.SetVar(match[1], match[2])

		logValue := match[2]
		if len(logValue) > 1024 {
			logValue = fmt.Sprintf("(%v bytes)", len(logValue))
		}

		t.logger.Infof("set variable %v: (string) %v", match[1], logValue)

		return true
	}

	match = outputJSONPattern.FindStringSubmatch(line)
	if match != nil {
		var varValue interface{}

		err := json.Unmarshal([]byte(match[2]), &varValue)
		if err != nil {
			t.logger.Warnf("error parsing ::set-var expression: %v", err.Error())
		} else {
			t.ctx.Vars.SetVar(match[1], varValue)
			t.logger.Infof("set variable %v: (json) %v", match[1], varValue)

			return true
		}
	}

	match = outputOutPattern.FindStringSubmatch(line)
	if match != nil {
		var varValue interface{}

		if match[2] == "-json" {
			err := json.Unmarshal([]byte(match[4]), &varValue)
			if err != nil {
				t.logger.Warnf("error parsing ::set-output-json expression: %v", err.Error())
			} else {
				t.ctx.Outputs.SetVar(match[3], varValue)
				t.logger.Infof("set output %v: (json) %v", match[3], varValue)

				return true
			}
		} else {
			t.ctx.Outputs.SetVar(match[3], match[4])

			logValue := match[4]
			if len(logValue) > 1024 {
				logValue = fmt.Sprintf("(%v bytes)", len(logValue))
			}

			t.logger.Infof("set output %v: (string) %v", match[3], logValue)
		}
	}

	return false
}
