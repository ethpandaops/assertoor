package runshell

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/jmoiron/sqlx"
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

	// create temp dir for task
	taskDir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("assertoor_%v_%v_", t.ctx.Scheduler.GetTestRunID(), t.ctx.Index))
	if err != nil {
		cmdLogger.Errorf("failed creating task dir: %v", err)
		return err
	}

	defer func() {
		if err2 := os.RemoveAll(taskDir); err2 != nil {
			cmdLogger.Errorf("failed cleaning up task dir: %v", err2)
		}
	}()

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

	// create summaries file
	summaryFile, err := newResultFile(filepath.Join(taskDir, "summary"))
	if err != nil {
		cmdLogger.Errorf("failed creating summary file: %v", err)
		return err
	}

	command.Env = append(command.Env, fmt.Sprintf("ASSERTOOR_SUMMARY=%v", summaryFile.FilePath()))

	// create folder for result files
	resultDir := filepath.Join(taskDir, "results")
	if err = os.MkdirAll(resultDir, 0o700); err != nil {
		cmdLogger.Errorf("failed creating result dir: %v", err)
		return err
	}

	command.Env = append(command.Env, fmt.Sprintf("ASSERTOOR_RESULT_DIR=%v", resultDir))

	defer func() {
		t.storeTaskResults(summaryFile, resultDir)
	}()

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

	if err := stdin.Close(); err != nil {
		t.logger.WithError(err).Warn("failed to close stdin")
	}

	// wait for process & output streams
	var execErr error

	waitChan := make(chan bool)
	go func() {
		defer close(waitChan)

		<-stdoutCloseChan
		<-stderrCloseChan

		execErr = command.Wait()
	}()

	// add context kill handler
	go func() {
		select {
		case <-ctx.Done():
			cmdLogger.Warn("sending SIGINT due to context cancellation")

			if err := command.Process.Signal(syscall.SIGINT); err != nil {
				cmdLogger.Warnf("failed sending SIGINT: %v", err)
			}

			select {
			case <-time.After(5 * time.Second):
				cmdLogger.Warn("killing command due to context timeout")

				if err := command.Process.Kill(); err != nil {
					cmdLogger.Warnf("failed killing command: %v", err)
				}
			case <-waitChan:
			}
		case <-waitChan:
		}
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
				t.logger.Infof("set output %v: (json)", match[3])

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

func (t *Task) storeTaskResults(summaryFile *resultFile, resultDir string) {
	// store files to db
	database := t.ctx.Scheduler.GetServices().Database()
	if err2 := database.RunTransaction(func(tx *sqlx.Tx) error {
		// store summary file
		data, err3 := summaryFile.Cleanup()
		if err3 != nil {
			t.logger.Errorf("failed cleaning up summary file: %v", err3)
		} else if len(data) > 0 {
			if err3 = database.UpsertTaskResult(tx, &db.TaskResult{
				RunID:  t.ctx.Scheduler.GetTestRunID(),
				TaskID: uint64(t.ctx.Index),
				Type:   "summary",
				Index:  0,
				Name:   "",
				Size:   uint64(len(data)),
				Data:   data,
			}); err3 != nil {
				t.logger.Errorf("failed storing summary file to db: %v", err3)
			}
		}

		// store result files
		fileIdx := uint64(0)

		var storeResultFilesFn func(path string, prefix string)
		storeResultFilesFn = func(path string, prefix string) {
			if prefix != "" {
				prefix += "/"
			}

			files, err3 := os.ReadDir(path)
			if err3 != nil {
				t.logger.Errorf("failed reading result dir: %v", err3)
			} else {
				for _, file := range files {
					if file.IsDir() {
						storeResultFilesFn(filepath.Join(path, file.Name()), fmt.Sprintf("%v%v", prefix, file.Name()))
						continue
					}

					data, err3 := os.ReadFile(filepath.Join(path, file.Name()))
					if err3 != nil {
						t.logger.Errorf("failed reading result file: %v", err3)
					} else if err3 = database.UpsertTaskResult(tx, &db.TaskResult{
						RunID:  t.ctx.Scheduler.GetTestRunID(),
						TaskID: uint64(t.ctx.Index),
						Type:   "result",
						Index:  fileIdx,
						Name:   fmt.Sprintf("%v%v", prefix, file.Name()),
						Size:   uint64(len(data)),
						Data:   data,
					}); err3 != nil {
						t.logger.Errorf("failed storing result file to db: %v", err3)
					}

					fileIdx++
				}
			}
		}

		storeResultFilesFn(resultDir, "")

		return nil
	}); err2 != nil {
		t.logger.Errorf("failed storing task results to db: %v", err2)
	}
}
