package runjavascript

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

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_javascript"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs a JavaScript snippet through Node.js. envVars are JSON-decoded and exposed both as process.env and as the 'env' object. Helpers: setOutput, setOutputJSON, setVar, setVarJSON, writeResultFile, writeSummary.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
		NewTask:     NewTask,
	}
)

// preamble is prepended to every user script. It:
//   - parses each env var (set from envVars config) as JSON when possible,
//     exposing the result as the global `env` object;
//   - exposes ASSERTOOR_SUMMARY / ASSERTOOR_RESULT_DIR as constants;
//   - provides `setOutput[JSON]`, `setVar[JSON]`, `writeResultFile`, and
//     `writeSummary` helpers that emit the same `::set-*` markers as
//     run_shell and write to the shared task dirs.
//
// The user's script is wrapped in `(async () => { ... })()` so top-level
// await works.
const preamble = `
const fs = require('fs');
const path = require('path');

const SUMMARY_FILE     = process.env.ASSERTOOR_SUMMARY     || '';
const RESULT_DIR       = process.env.ASSERTOOR_RESULT_DIR  || '';
const TEST_RESULT_FILE = process.env.ASSERTOOR_TEST_RESULT || '';
const __envKeys        = (process.env.__ASSERTOOR_ENV_KEYS || '').split(',').filter(Boolean);

const env = {};
for (const k of __envKeys) {
  const raw = process.env[k];
  if (raw === undefined) continue;
  try { env[k] = JSON.parse(raw); }
  catch { env[k] = raw; }
}

function setVar(name, value)      { console.log('::set-var '      + name + ' ' + String(value)); }
function setVarJSON(name, value)   { console.log('::set-json '     + name + ' ' + JSON.stringify(value)); }
function setOutput(name, value)    { console.log('::set-output '   + name + ' ' + String(value)); }
function setOutputJSON(name, value){ console.log('::set-output-json ' + name + ' ' + JSON.stringify(value)); }

function writeResultFile(name, content) {
  if (!RESULT_DIR) throw new Error('ASSERTOOR_RESULT_DIR is not set');
  const dest = path.join(RESULT_DIR, name);
  fs.mkdirSync(path.dirname(dest), { recursive: true });
  fs.writeFileSync(dest, content);
}
function writeSummary(content) {
  if (!SUMMARY_FILE) throw new Error('ASSERTOOR_SUMMARY is not set');
  fs.writeFileSync(SUMMARY_FILE, content);
}

// writeTestResult and appendTestResult target the shared per-test-run
// markdown file. Anything written here shows up on the run page's
// Result panel (one centralised, easy-to-find place across all tasks).
function writeTestResult(content) {
  if (!TEST_RESULT_FILE) throw new Error('ASSERTOOR_TEST_RESULT is not set');
  fs.writeFileSync(TEST_RESULT_FILE, content);
}
function appendTestResult(content) {
  if (!TEST_RESULT_FILE) throw new Error('ASSERTOOR_TEST_RESULT is not set');
  fs.appendFileSync(TEST_RESULT_FILE, content);
}

(async () => {
%s
})().catch(err => {
  console.error(err && err.stack ? err.stack : String(err));
  process.exit(1);
});
`

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
	scriptLogger := t.logger.WithField("interpreter", t.config.NodePath)
	scriptLogger.Info("running javascript")
	t.ctx.ReportProgress(0, "Running JavaScript snippet...")

	// create temp dir for task
	taskDir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("assertoor_js_%v_%v_", t.ctx.Scheduler.GetTestRunID(), t.ctx.Index))
	if err != nil {
		scriptLogger.Errorf("failed creating task dir: %v", err)

		return err
	}

	defer func() {
		if err2 := os.RemoveAll(taskDir); err2 != nil {
			scriptLogger.Errorf("failed cleaning up task dir: %v", err2)
		}
	}()

	// write the (preamble + user script) to a file so node can give us
	// meaningful stack traces with line numbers.
	scriptPath := filepath.Join(taskDir, "script.cjs")
	scriptSource := fmt.Sprintf(preamble, t.config.Script)

	if err = os.WriteFile(scriptPath, []byte(scriptSource), 0o600); err != nil {
		scriptLogger.Errorf("failed writing script file: %v", err)

		return err
	}

	nodeArgs := append([]string{}, t.config.NodeArgs...)
	nodeArgs = append(nodeArgs, scriptPath)

	command := exec.CommandContext(ctx, t.config.NodePath, nodeArgs...) //nolint:gosec // user-provided script is the whole point

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

	// Assemble all process env vars (user envVars, ASSERTOOR_* paths, PATH).
	summaryFile, resultDir, envErr := t.buildCommandEnv(taskDir, command, scriptLogger)
	if envErr != nil {
		return envErr
	}

	defer func() {
		t.storeTaskResults(summaryFile, resultDir)
	}()

	// start node
	if err = command.Start(); err != nil {
		scriptLogger.Errorf("failed starting node: %v", err)

		return err
	}

	// We don't actually feed anything via stdin (the script is on disk);
	// close stdin so any read in the user script returns EOF cleanly.
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
			scriptLogger.Warn("sending SIGINT due to context cancellation")

			if err := command.Process.Signal(syscall.SIGINT); err != nil {
				scriptLogger.Warnf("failed sending SIGINT: %v", err)
			}

			select {
			case <-time.After(5 * time.Second):
				scriptLogger.Warn("killing node process due to context timeout")

				if err := command.Process.Kill(); err != nil {
					scriptLogger.Warnf("failed killing node process: %v", err)
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
				scriptLogger.Infof("OUT: %v", line)
			}
		case line := <-stderrChan:
			scriptLogger.Warnf("ERR: %v", line)
		case <-waitChan:
			break cmdloop
		}
	}

	// await completion
	if execErr != nil {
		scriptLogger.Errorf("javascript execution failed: %v", execErr)

		return execErr
	}

	scriptLogger.Info("javascript completed successfully")
	t.ctx.ReportProgress(100, "JavaScript completed")

	return nil
}

// buildCommandEnv assembles the child process's environment:
//   - one var per entry in t.config.EnvVars (JSON-encoded resolved value)
//   - __ASSERTOOR_ENV_KEYS so the preamble knows which keys to JSON.parse
//   - PATH inherited from the parent
//   - ASSERTOOR_SUMMARY (newly-created file)
//   - ASSERTOOR_RESULT_DIR (newly-created dir)
//   - ASSERTOOR_TEST_RESULT (shared per-test-run file, best-effort)
//
// It returns the summary-file wrapper and the result-dir path the caller
// uses to persist task results, or an error for unrecoverable setup
// failures.
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

	command.Env = append(command.Env, fmt.Sprintf("__ASSERTOOR_ENV_KEYS=%s", joinComma(envKeys)))

	for _, kv := range os.Environ() {
		if len(kv) > 5 && kv[:5] == "PATH=" {
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

	// Shared per-test-run markdown file. Anything the script writes to
	// this path is persisted as the run-level Result panel content.
	if testResultPath, terr := t.ctx.Scheduler.TestResultPath(); terr == nil {
		command.Env = append(command.Env, fmt.Sprintf("ASSERTOOR_TEST_RESULT=%v", testResultPath))
	} else {
		scriptLogger.WithError(terr).Warn("failed to obtain test-result path; ASSERTOOR_TEST_RESULT will be unset")
	}

	return summaryFile, resultDir, nil
}

func joinComma(xs []string) string {
	out := ""

	for i, x := range xs {
		if i > 0 {
			out += ","
		}

		out += x
	}

	return out
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
		var varValue interface{}
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
			var varValue interface{}
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
		// store summary file (only if non-empty)
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
