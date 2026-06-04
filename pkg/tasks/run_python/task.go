package runpython

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
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/helper"
	runpythonuv "github.com/ethpandaops/assertoor/pkg/tasks/run_python_uv"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_python"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs a Python snippet. envVars are JSON-decoded and exposed both as os.environ and as the 'env' dict. When useUv is true (default), the script runs inside the uv-managed venv referenced by 'venvVar'; if the variable is unset, an empty venv is auto-initialized and a cleanup task is registered. Helpers: set_output, set_output_json, set_var, set_var_json, write_result_file, write_summary, write_test_result, append_test_result.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
		NewTask:     NewTask,
	}
)

// autoInitMutexes serializes the auto-init path across concurrent
// run_python tasks within the same test run. Without it, two siblings
// firing at once would both decide "venv var is empty, must create one"
// and stomp on each other.
var (
	autoInitMutexes     = map[uint64]*sync.Mutex{}
	autoInitMutexesLock sync.Mutex
)

func autoInitMutex(testRunID uint64) *sync.Mutex {
	autoInitMutexesLock.Lock()
	defer autoInitMutexesLock.Unlock()

	if m, ok := autoInitMutexes[testRunID]; ok {
		return m
	}

	m := &sync.Mutex{}
	autoInitMutexes[testRunID] = m

	return m
}

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

func (t *Task) Config() any {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	if err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars); err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	pythonBin, err := t.resolvePython(ctx)
	if err != nil {
		return err
	}

	scriptLogger := t.logger.WithField("interpreter", pythonBin)
	scriptLogger.Info("running python")
	t.ctx.ReportProgress(0, "Running Python snippet...")

	taskDir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("assertoor_py_%v_%v_", t.ctx.Scheduler.GetTestRunID(), t.ctx.Index))
	if err != nil {
		scriptLogger.Errorf("failed creating task dir: %v", err)

		return err
	}

	defer func() {
		if err2 := os.RemoveAll(taskDir); err2 != nil {
			scriptLogger.Errorf("failed cleaning up task dir: %v", err2)
		}
	}()

	scriptPath := filepath.Join(taskDir, "script.py")
	scriptSource := fmt.Sprintf(preamble, indentScript(t.config.Script, "    "))

	if err = os.WriteFile(scriptPath, []byte(scriptSource), 0o600); err != nil {
		scriptLogger.Errorf("failed writing script file: %v", err)

		return err
	}

	pythonArgs := make([]string, 0, len(t.config.PythonArgs)+1)
	pythonArgs = append(pythonArgs, t.config.PythonArgs...)
	pythonArgs = append(pythonArgs, scriptPath)

	command := exec.CommandContext(ctx, pythonBin, pythonArgs...)

	stdin, err := command.StdinPipe()
	if err != nil {
		scriptLogger.Errorf("failed getting stdin pipe")

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

	stdoutChan, stdoutCloseChan := t.readOutputStream(stdout, scriptLogger.WithField("stream", "stdout"))
	defer close(stdoutChan)

	stderrChan, stderrCloseChan := t.readOutputStream(stderr, scriptLogger.WithField("stream", "stderr"))
	defer close(stderrChan)

	summaryFile, resultDir, envErr := t.buildCommandEnv(taskDir, command, scriptLogger)
	if envErr != nil {
		return envErr
	}

	defer func() {
		t.storeTaskResults(summaryFile, resultDir)
	}()

	if err = command.Start(); err != nil {
		scriptLogger.Errorf("failed starting python: %v", err)

		return err
	}

	if err := stdin.Close(); err != nil {
		t.logger.WithError(err).Warn("failed to close stdin")
	}

	var execErr error

	waitChan := make(chan bool)

	go func() {
		defer close(waitChan)

		<-stdoutCloseChan
		<-stderrCloseChan

		execErr = command.Wait()
	}()

	go func() {
		select {
		case <-ctx.Done():
			scriptLogger.Warn("sending SIGINT due to context cancellation")

			if err := command.Process.Signal(syscall.SIGINT); err != nil {
				scriptLogger.Warnf("failed sending SIGINT: %v", err)
			}

			select {
			case <-time.After(5 * time.Second):
				scriptLogger.Warn("killing python process due to context timeout")

				if err := command.Process.Kill(); err != nil {
					scriptLogger.Warnf("failed killing python process: %v", err)
				}
			case <-waitChan:
			}
		case <-waitChan:
		}
	}()

cmdloop:
	for {
		select {
		case line := <-stdoutChan:
			if !t.parseOutputVars(line) {
				scriptLogger.Infof("OUT: %v", line)
			}
		case line := <-stderrChan:
			scriptLogger.Warnf("ERR: %v", line)
		case <-waitChan:
			break cmdloop
		}
	}

	if execErr != nil {
		scriptLogger.Errorf("python execution failed: %v", execErr)

		return execErr
	}

	scriptLogger.Info("python completed successfully")
	t.ctx.ReportProgress(100, "Python completed")

	return nil
}

// resolvePython returns the path to the python binary to invoke. When
// useUv is true, it consults the venv variable; if unset, it auto-inits
// a default venv via run_python_uv and registers a cleanup task. When
// useUv is false or auto-init fails, it falls back to pythonPath.
func (t *Task) resolvePython(ctx context.Context) (string, error) {
	if !t.config.UseUV {
		return t.config.PythonPath, nil
	}

	if path := t.lookupVenvPython(); path != "" {
		if err := t.ensureRequirements(ctx, path); err != nil {
			return "", err
		}

		return path, nil
	}

	// auto-init under a per-test mutex so concurrent siblings don't race.
	mu := autoInitMutex(t.ctx.Scheduler.GetTestRunID())
	mu.Lock()
	defer mu.Unlock()

	// re-check after acquiring the lock: another sibling may have just
	// finished auto-init.
	if path := t.lookupVenvPython(); path != "" {
		if err := t.ensureRequirements(ctx, path); err != nil {
			return "", err
		}

		return path, nil
	}

	if err := t.autoInitVenv(); err != nil {
		t.logger.WithError(err).Warn("auto-init of uv venv failed; falling back to system python")

		return t.config.PythonPath, nil
	}

	path := t.lookupVenvPython()
	if path == "" {
		t.logger.Warn("auto-init did not populate venv var; falling back to system python")

		return t.config.PythonPath, nil
	}

	if err := t.ensureRequirements(ctx, path); err != nil {
		return "", err
	}

	return path, nil
}

func (t *Task) lookupVenvPython() string {
	val, found := t.ctx.Vars.LookupVar(t.config.VenvVar)
	if !found {
		return ""
	}

	venvPath, ok := val.(string)
	if !ok || venvPath == "" {
		return ""
	}

	return filepath.Join(venvPath, "bin", "python")
}

// autoInitVenv delegates to a synthetic run_python_uv task. We build it
// inline with helper.NewRawMessage so the same config validation and
// cleanup-registration code path runs.
func (t *Task) autoInitVenv() error {
	if t.ctx.NewTask == nil {
		return fmt.Errorf("NewTask not available on TaskContext")
	}

	cfg := runpythonuv.DefaultConfig()
	cfg.UVPath = t.config.UVPath
	cfg.VenvVar = t.config.VenvVar

	opts := &types.TaskOptions{
		Name:    runpythonuv.TaskName,
		Title:   fmt.Sprintf("auto-init uv venv (%s)", t.config.VenvVar),
		Config:  helper.NewRawMessage(cfg),
		Timeout: helper.Duration{Duration: 5 * time.Minute},
	}

	_, err := t.ctx.NewTask(opts, t.ctx.Vars)

	return err
}

func (t *Task) ensureRequirements(ctx context.Context, pythonBin string) error {
	if len(t.config.Requirements) == 0 {
		return nil
	}

	args := make([]string, 0, 4+len(t.config.Requirements))
	args = append(args, "pip", "install", "--python", pythonBin)
	args = append(args, t.config.Requirements...)

	cmd := exec.CommandContext(ctx, t.config.UVPath, args...) //nolint:gosec // uvPath and requirements come from user config

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		t.logger.WithField("step", "uv pip install").Info(string(output))
	}

	if err != nil {
		return fmt.Errorf("uv pip install failed: %w", err)
	}

	return nil
}

func (t *Task) buildCommandEnv(
	taskDir string,
	command *exec.Cmd,
	scriptLogger logrus.FieldLogger,
) (*resultFile, string, error) {
	envKeys := make([]string, 0, len(t.config.EnvVars))

	for envName, varName := range t.config.EnvVars {
		varValue, varFound, err := t.ctx.Vars.ResolveQuery(varName)
		if err != nil {
			scriptLogger.Errorf("failed parsing var query for env variable %v: %v", envName, err)

			return nil, "", err
		}

		if !varFound {
			continue
		}

		varJSON, err := json.Marshal(varValue)
		if err != nil {
			scriptLogger.Errorf("failed encoding env variable %v: %v", envName, err)

			return nil, "", err
		}

		command.Env = append(command.Env, fmt.Sprintf("%v=%v", envName, string(varJSON)))
		envKeys = append(envKeys, envName)
	}

	command.Env = append(command.Env, fmt.Sprintf("__ASSERTOOR_ENV_KEYS=%s", strings.Join(envKeys, ",")))

	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") {
			command.Env = append(command.Env, kv)
		}
	}

	summaryFile, err := newResultFile(filepath.Join(taskDir, "summary"))
	if err != nil {
		scriptLogger.Errorf("failed creating summary file: %v", err)

		return nil, "", err
	}

	command.Env = append(command.Env, fmt.Sprintf("ASSERTOOR_SUMMARY=%v", summaryFile.FilePath()))

	resultDir := filepath.Join(taskDir, "results")
	if err := os.MkdirAll(resultDir, 0o700); err != nil {
		scriptLogger.Errorf("failed creating result dir: %v", err)

		return nil, "", err
	}

	command.Env = append(command.Env, fmt.Sprintf("ASSERTOOR_RESULT_DIR=%v", resultDir))

	if testResultPath, terr := t.ctx.Scheduler.TestResultPath(); terr == nil {
		command.Env = append(command.Env, fmt.Sprintf("ASSERTOOR_TEST_RESULT=%v", testResultPath))
	} else {
		scriptLogger.WithError(terr).Warn("failed to obtain test-result path; ASSERTOOR_TEST_RESULT will be unset")
	}

	return summaryFile, resultDir, nil
}

func indentScript(src, indent string) string {
	if src == "" {
		// Empty body would be a syntax error inside async def; emit a no-op pass.
		return indent + "pass"
	}

	lines := strings.Split(src, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}

		lines[i] = indent + line
	}

	return strings.Join(lines, "\n")
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

var (
	outputVarPattern  = regexp.MustCompile(`^::set-var +([^ ]+) +(.*)$`)
	outputJSONPattern = regexp.MustCompile(`^::set-json +([^ ]+) +(.*)$`)
	outputOutPattern  = regexp.MustCompile(`^::set-out(put)?(-json)? +([^ ]+) +(.*)$`)
)

func (t *Task) parseOutputVars(line string) bool {
	if match := outputVarPattern.FindStringSubmatch(line); match != nil {
		t.ctx.Vars.SetVar(match[1], match[2])

		logValue := match[2]
		if len(logValue) > 1024 {
			logValue = fmt.Sprintf("(%v bytes)", len(logValue))
		}

		t.logger.Infof("set variable %v: (string) %v", match[1], logValue)

		return true
	}

	if match := outputJSONPattern.FindStringSubmatch(line); match != nil {
		var varValue any
		if err := json.Unmarshal([]byte(match[2]), &varValue); err != nil {
			t.logger.Warnf("error parsing ::set-json expression: %v", err.Error())
		} else {
			t.ctx.Vars.SetVar(match[1], varValue)
			t.logger.Infof("set variable %v: (json) %v", match[1], varValue)

			return true
		}
	}

	if match := outputOutPattern.FindStringSubmatch(line); match != nil {
		if match[2] == "-json" {
			var varValue any
			if err := json.Unmarshal([]byte(match[4]), &varValue); err != nil {
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
	database := t.ctx.Scheduler.GetServices().Database()

	if err2 := database.RunTransaction(func(tx *sqlx.Tx) error {
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

		fileIdx := uint64(0)

		var storeResultFilesFn func(path string, prefix string)

		storeResultFilesFn = func(path string, prefix string) {
			if prefix != "" {
				prefix += "/"
			}

			files, err3 := os.ReadDir(path)
			if err3 != nil {
				t.logger.Errorf("failed reading result dir: %v", err3)

				return
			}

			for _, file := range files {
				if file.IsDir() {
					storeResultFilesFn(filepath.Join(path, file.Name()), fmt.Sprintf("%v%v", prefix, file.Name()))

					continue
				}

				data, err4 := os.ReadFile(filepath.Join(path, file.Name()))
				if err4 != nil {
					t.logger.Errorf("failed reading result file: %v", err4)

					continue
				}

				if err4 = database.UpsertTaskResult(tx, &db.TaskResult{
					RunID:  t.ctx.Scheduler.GetTestRunID(),
					TaskID: uint64(t.ctx.Index),
					Type:   "result",
					Index:  fileIdx,
					Name:   fmt.Sprintf("%v%v", prefix, file.Name()),
					Size:   uint64(len(data)),
					Data:   data,
				}); err4 != nil {
					t.logger.Errorf("failed storing result file to db: %v", err4)
				}

				fileIdx++
			}
		}

		storeResultFilesFn(resultDir, "")

		return nil
	}); err2 != nil {
		t.logger.Errorf("failed storing task results to db: %v", err2)
	}
}
