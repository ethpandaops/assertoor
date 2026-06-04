package runpythonuv

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
	runshell "github.com/ethpandaops/assertoor/pkg/tasks/run_shell"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_python_uv"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Initializes a uv-managed Python virtual environment shared by subsequent run_python tasks in the same test. The venv path is exposed as a runtime variable; a cleanup task is registered to delete it when the test ends.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
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
	t.ctx.ReportProgress(0, "Initializing uv venv...")

	// Idempotency: if SkipIfSet and the venv var is already populated, do
	// nothing. This lets run_python and run_python_uv coexist without
	// duplicate setup.
	if t.config.SkipIfSet {
		if existing, found := t.ctx.Vars.LookupVar(t.config.VenvVar); found {
			if pathStr, ok := existing.(string); ok && pathStr != "" {
				t.logger.WithField("path", pathStr).Info("venv already initialized; skipping")
				t.ctx.ReportProgress(100, "venv already initialized")

				return nil
			}
		}
	}

	venvDir, err := os.MkdirTemp(os.TempDir(), fmt.Sprintf("assertoor_pyuv_%v_", t.ctx.Scheduler.GetTestRunID()))
	if err != nil {
		return fmt.Errorf("failed to create venv temp dir: %w", err)
	}

	// Inside the temp dir, uv venv writes the venv files directly. Use a
	// nested 'venv' subdir so we can keep auxiliary artifacts (e.g. logs)
	// next to it later if needed.
	venvPath := filepath.Join(venvDir, "venv")

	if err := t.runUVVenv(ctx, venvPath); err != nil {
		// Best-effort cleanup since the cleanup task hasn't been registered yet.
		os.RemoveAll(venvDir)

		return err
	}

	t.ctx.ReportProgress(50, "Installing requirements...")

	if err := t.installRequirements(ctx, venvPath); err != nil {
		os.RemoveAll(venvDir)

		return err
	}

	t.ctx.Vars.SetVar(t.config.VenvVar, venvPath)
	t.logger.WithField("path", venvPath).WithField("var", t.config.VenvVar).Info("uv venv ready")

	if err := t.registerCleanup(venvDir); err != nil {
		// Cleanup registration failure is recoverable - log and continue. The
		// temp dir will eventually be reaped by the OS.
		t.logger.WithError(err).Warn("failed to register cleanup task; venv temp dir will be left behind")
	}

	t.ctx.ReportProgress(100, "uv venv ready")

	return nil
}

func (t *Task) runUVVenv(ctx context.Context, venvPath string) error {
	args := []string{"venv"}
	if t.config.PythonVersion != "" {
		args = append(args, "--python", t.config.PythonVersion)
	}

	args = append(args, venvPath)

	return t.runUVCommand(ctx, args, "uv venv")
}

func (t *Task) installRequirements(ctx context.Context, venvPath string) error {
	if len(t.config.Requirements) == 0 {
		return nil
	}

	pythonBin := filepath.Join(venvPath, "bin", "python")

	args := make([]string, 0, 4+len(t.config.Requirements))
	args = append(args, "pip", "install", "--python", pythonBin)
	args = append(args, t.config.Requirements...)

	return t.runUVCommand(ctx, args, "uv pip install")
}

func (t *Task) runUVCommand(ctx context.Context, args []string, what string) error {
	cmd := exec.CommandContext(ctx, t.config.UVPath, args...) //nolint:gosec // uvPath and requirements come from user config; running them is the whole point

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		t.logger.WithField("step", what).Info(string(output))
	}

	if err != nil {
		return fmt.Errorf("%s failed: %w", what, err)
	}

	return nil
}

// registerCleanup queues a cleanup task that removes the venv temp dir.
// PrependCleanupTask (wired into ctx.AddCleanupTask) puts it ahead of any
// previously-registered cleanup task, giving LIFO teardown order.
func (t *Task) registerCleanup(venvDir string) error {
	if t.ctx.AddCleanupTask == nil {
		return fmt.Errorf("AddCleanupTask not available on TaskContext")
	}

	cleanupCfg := runshell.Config{
		Shell:   "bash",
		Command: fmt.Sprintf("rm -rf %q", venvDir),
	}

	opts := &types.TaskOptions{
		Name:    runshell.TaskName,
		Title:   fmt.Sprintf("cleanup uv venv (%s)", t.config.VenvVar),
		Config:  helper.NewRawMessage(cleanupCfg),
		Timeout: helper.Duration{Duration: 60 * time.Second},
	}

	_, err := t.ctx.AddCleanupTask(opts)

	return err
}
