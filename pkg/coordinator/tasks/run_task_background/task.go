package runtaskbackground

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_task_background"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs foreground and background task with configurable dependencies.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx            *types.TaskContext
	options        *types.TaskOptions
	config         Config
	logger         logrus.FieldLogger
	foregroundTask types.Task
	backgroundTask types.Task
	resultChanMtx  sync.Mutex
	resultChan     chan taskResult
	foregroundChan chan bool
}

type taskResult struct {
	result types.TaskResult
	err    error
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
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
	if err2 := config.Validate(); err2 != nil {
		return err2
	}

	// init background task
	if config.BackgroundTask != nil {
		bgTaskOpts, err2 := t.ctx.Scheduler.ParseTaskOptions(config.BackgroundTask)
		if err2 != nil {
			return fmt.Errorf("failed parsing background task config: %w", err2)
		}

		t.backgroundTask, err = t.ctx.NewTask(bgTaskOpts, t.ctx.Vars.NewScope())
		if err != nil {
			return fmt.Errorf("failed initializing background task: %w", err)
		}
	}

	// init foreground task
	fgTaskOpts, err := t.ctx.Scheduler.ParseTaskOptions(config.ForegroundTask)
	if err != nil {
		return fmt.Errorf("failed parsing foreground task config: %w", err)
	}

	taskVars := t.ctx.Vars
	if config.NewVariableScope {
		taskVars = taskVars.NewScope()
	}

	t.foregroundTask, err = t.ctx.NewTask(fgTaskOpts, taskVars)
	if err != nil {
		return fmt.Errorf("failed initializing foreground task: %w", err)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	t.resultChan = make(chan taskResult, 10)
	t.foregroundChan = make(chan bool)

	childCtx, cancel := context.WithCancel(ctx)

	if t.backgroundTask != nil {
		go t.execBackgroundTask(childCtx)
	}

	go t.execForegroundTask(childCtx)

	result := <-t.resultChan
	t.ctx.SetResult(result.result)

	t.resultChanMtx.Lock()
	t.resultChan = nil
	t.resultChanMtx.Unlock()
	cancel()

	<-t.foregroundChan

	return result.err
}

func (t *Task) completeWithResult(result types.TaskResult, err error) {
	t.resultChanMtx.Lock()
	defer t.resultChanMtx.Unlock()

	if t.resultChan == nil {
		return
	}

	t.resultChan <- taskResult{
		result: result,
		err:    err,
	}
}

func (t *Task) execBackgroundTask(ctx context.Context) {
	err := t.ctx.Scheduler.ExecuteTask(ctx, t.backgroundTask, nil)

	if ctx.Err() != nil {
		return
	}

	taskResult := t.ctx.Scheduler.GetTaskStatus(t.backgroundTask)

	//nolint:goconst // ignore
	taskStatus := "success"
	if taskResult.Result == types.TaskResultFailure {
		taskStatus = "failure"
	}

	t.logger.Infof("background task completed. status: %v, err: %v", taskStatus, err)

	switch t.config.OnBackgroundComplete {
	case "fail":
		t.completeWithResult(types.TaskResultFailure, fmt.Errorf("background task completed unexpectedly"))
	case "success", "succeed":
		t.completeWithResult(types.TaskResultSuccess, nil)
	case "failOrIgnore":
		if taskResult.Result == types.TaskResultFailure {
			t.completeWithResult(types.TaskResultFailure, fmt.Errorf("background task completed with failure"))
		}
	}
}

func (t *Task) execForegroundTask(ctx context.Context) {
	defer func() {
		close(t.foregroundChan)
	}()

	err := t.ctx.Scheduler.ExecuteTask(ctx, t.foregroundTask, func(ctx context.Context, cancelFn context.CancelFunc, _ types.Task) {
		t.watchTaskResult(ctx, cancelFn)
	})

	taskResult := t.ctx.Scheduler.GetTaskStatus(t.foregroundTask)

	taskStatus := "success"
	if taskResult.Result == types.TaskResultFailure {
		taskStatus = "failure"
	}

	t.logger.Infof("foreground task completed. status: %v, err: %v", taskStatus, err)

	t.completeWithResult(taskResult.Result, taskResult.Error)
}

func (t *Task) watchTaskResult(ctx context.Context, _ context.CancelFunc) {
	currentResult := types.TaskResultNone

	for {
		updateChan := t.ctx.Scheduler.GetTaskResultUpdateChan(t.foregroundTask, currentResult)
		if updateChan != nil {
			select {
			case <-ctx.Done():
				return
			case <-updateChan:
			}
		}

		taskStatus := t.ctx.Scheduler.GetTaskStatus(t.foregroundTask)
		if taskStatus.Result == currentResult {
			continue
		}

		currentResult = taskStatus.Result

		if t.config.ExitOnForegroundSuccess && currentResult == types.TaskResultSuccess {
			t.logger.Infof("foreground task succeeded. stopping task with success result")
			t.completeWithResult(types.TaskResultSuccess, nil)

			return
		}

		if t.config.ExitOnForegroundFailure && currentResult == types.TaskResultFailure {
			t.logger.Infof("foreground task failed. stopping task with failure result")
			t.completeWithResult(types.TaskResultFailure, taskStatus.Error)

			return
		}

		t.ctx.SetResult(currentResult)
	}
}
