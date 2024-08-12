package runexternaltasks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	TaskName       = "run_external_tasks"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Run external test playbook.",
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
	// load test yaml file
	testConfig, err := t.loadTestConfig(ctx, t.config.TestFile)
	if err != nil {
		return err
	}

	// create new variable scope for test configuration
	testVars := t.ctx.Vars.NewScope()
	testVars.SetVar("scopeOwner", uint64(t.ctx.Index))
	t.ctx.Outputs.SetSubScope("childScope", testVars)

	// add default config from external test to variable scope
	for k, v := range testConfig.Config {
		testVars.SetDefaultVar(k, v)
	}

	// add custom config supplied via this task to variable scope
	for k, v := range t.config.TestConfig {
		testVars.SetVar(k, v)
	}

	// merge configVars mappings & copy varibles to scope
	for k, v := range t.config.TestConfigVars {
		testConfig.ConfigVars[k] = v
	}

	err = testVars.CopyVars(testVars, testConfig.ConfigVars)
	if err != nil {
		return fmt.Errorf("error decoding external test configVars %v: %v", t.config.TestFile, err)
	}

	// init child tasks
	tasks := []types.TaskIndex{}

	for i := range testConfig.Tasks {
		taskOptions, err := t.ctx.Scheduler.ParseTaskOptions(&testConfig.Tasks[i])
		if err != nil {
			return err
		}

		taskIndex, err := t.ctx.NewTask(taskOptions, testVars)
		if err != nil {
			return err
		}

		tasks = append(tasks, taskIndex)
	}

	// init cleanup tasks
	cleanupTasks := []types.TaskIndex{}

	for i := range testConfig.CleanupTasks {
		taskOptions, err := t.ctx.Scheduler.ParseTaskOptions(&testConfig.CleanupTasks[i])
		if err != nil {
			return err
		}

		taskIndex, err := t.ctx.NewTask(taskOptions, testVars)
		if err != nil {
			return err
		}

		cleanupTasks = append(cleanupTasks, taskIndex)
	}

	// execute child tasks
	var resError error

taskLoop:
	for i, task := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := t.ctx.Scheduler.ExecuteTask(ctx, task, t.ctx.Scheduler.WatchTaskPass)

		switch {
		case t.config.IgnoreFailure:
			if err != nil {
				t.logger.Warnf("child task #%v failed: %w", i+1, err)
			}
		default:
			if err != nil {
				resError = fmt.Errorf("child task #%v failed: %w", i+1, err)
				break taskLoop
			}
		}
	}

	// execute cleanup tasks
	for i, task := range cleanupTasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := t.ctx.Scheduler.ExecuteTask(ctx, task, t.ctx.Scheduler.WatchTaskPass)
		if err != nil {
			t.logger.Warnf("cleanup task #%v failed: %w", i+1, err)
		}
	}

	if t.config.ExpectFailure {
		if resError == nil {
			return fmt.Errorf("test should have failed, but succeeded")
		}

		return nil
	}

	return resError
}

func (t *Task) loadTestConfig(ctx context.Context, testFile string) (*types.TestConfig, error) {
	var reader io.Reader

	if strings.HasPrefix(testFile, "http://") || strings.HasPrefix(testFile, "https://") {
		client := &http.Client{Timeout: time.Second * 120}

		req, err := http.NewRequestWithContext(ctx, "GET", testFile, http.NoBody)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("error loading test config from url: %v, result: %v %v", testFile, resp.StatusCode, resp.Status)
		}

		reader = resp.Body
	} else {
		f, err := os.Open(testFile)
		if err != nil {
			return nil, fmt.Errorf("error loading test config from file %v: %w", testFile, err)
		}

		defer f.Close()

		reader = f
	}

	decoder := yaml.NewDecoder(reader)
	testConfig := &types.TestConfig{}

	err := decoder.Decode(testConfig)
	if err != nil {
		return nil, fmt.Errorf("error decoding external test config %v: %v", testFile, err)
	}

	if testConfig.Config == nil {
		testConfig.Config = map[string]interface{}{}
	}

	if testConfig.ConfigVars == nil {
		testConfig.ConfigVars = map[string]string{}
	}

	return testConfig, nil
}
