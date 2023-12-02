package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/helper"
	"github.com/ethpandaops/minccino/pkg/coordinator/task/tasks"
	"github.com/ethpandaops/minccino/pkg/coordinator/task/types"
	"github.com/sirupsen/logrus"
)

type TaskScheduler struct {
	logger           logrus.FieldLogger
	taskCount        uint64
	allTasks         []types.Task
	rootTasks        []types.Task
	allCleanupTasks  []types.Task
	rootCleanupTasks []types.Task
	runningMutex     sync.Mutex
	runningTasks     map[types.Task]*types.TaskStatus
}

func NewTaskScheduler(logger logrus.FieldLogger) *TaskScheduler {
	return &TaskScheduler{
		logger:       logger,
		rootTasks:    make([]types.Task, 0),
		allTasks:     make([]types.Task, 0),
		runningTasks: make(map[types.Task]*types.TaskStatus),
	}
}

func (ts *TaskScheduler) ParseTaskOptions(rawtask *helper.RawMessage) (*types.TaskOptions, error) {
	options := &types.TaskOptions{}
	if err := rawtask.Unmarshal(&options); err != nil {
		return nil, fmt.Errorf("error parsing task: %w", err)
	}
	return options, nil
}

func (ts *TaskScheduler) AddRootTask(options *types.TaskOptions) (types.Task, error) {
	task, err := ts.newTask(options, nil, false)
	if err != nil {
		return nil, err
	}
	ts.rootTasks = append(ts.rootTasks, task)
	return task, nil
}

func (ts *TaskScheduler) AddCleanupTask(options *types.TaskOptions) (types.Task, error) {
	task, err := ts.newTask(options, nil, true)
	if err != nil {
		return nil, err
	}
	ts.rootCleanupTasks = append(ts.rootCleanupTasks, task)
	return task, nil
}

func (ts *TaskScheduler) GetTaskCount() int {
	return len(ts.allTasks)
}

func (ts *TaskScheduler) newTask(options *types.TaskOptions, parent types.Task, isCleanupTask bool) (types.Task, error) {
	// lookup task by name
	var taskDescriptor *types.TaskDescriptor
	for _, taskDesc := range tasks.AvailableTaskDescriptors {
		if taskDesc.Name == options.Name {
			taskDescriptor = taskDesc
			break
		}
	}
	if taskDescriptor == nil {
		return nil, fmt.Errorf("unknown task name: %v", options.Name)
	}

	// create instance of task
	var task types.Task
	taskCtx := &types.TaskContext{
		Scheduler:  ts,
		Index:      ts.taskCount,
		Logger:     ts.logger,
		ParentTask: parent,
		NewTask: func(options *types.TaskOptions) (types.Task, error) {
			return ts.newTask(options, task, isCleanupTask)
		},
	}
	ts.taskCount++
	var err error
	task, err = taskDescriptor.NewTask(taskCtx, options)
	if err != nil {
		return nil, fmt.Errorf("failed task '%v' initialization: %w", options.Name, err)
	}

	if isCleanupTask {
		ts.allCleanupTasks = append(ts.allCleanupTasks, task)
	} else {
		ts.allTasks = append(ts.allTasks, task)
	}
	return task, nil
}

func (ts *TaskScheduler) ValidateTaskConfigs() error {
	for _, task := range ts.allTasks {
		err := task.ValidateConfig()
		if err != nil {
			task.Logger().WithError(err).Errorf("config validation failed")
			return fmt.Errorf("task %v config validation failed: %w", task.Name(), err)
		}
	}
	for _, task := range ts.allCleanupTasks {
		err := task.ValidateConfig()
		if err != nil {
			task.Logger().WithError(err).Errorf("config validation failed")
			return fmt.Errorf("cleanup task %v config validation failed: %w", task.Name(), err)
		}
	}
	return nil
}

func (ts *TaskScheduler) RunTasks(ctx context.Context, timeout time.Duration) error {
	defer ts.runCleanupTasks(ctx)

	var tasksCtx context.Context
	if timeout > 0 {
		c, cancel := context.WithTimeout(ctx, timeout)
		tasksCtx = c
		defer cancel()
	} else {
		c, cancel := context.WithCancel(ctx)
		tasksCtx = c
		defer cancel()
	}

	for _, task := range ts.rootTasks {
		err := ts.ExecuteTask(tasksCtx, task, ts.WatchTaskPass)
		if err != nil {
			return err
		}
		if tasksCtx.Err() != nil {
			return tasksCtx.Err()
		}
	}
	return nil
}

func (ts *TaskScheduler) runCleanupTasks(ctx context.Context) {
	for _, task := range ts.rootCleanupTasks {
		if ctx.Err() != nil {
			return
		}
		ts.ExecuteTask(ctx, task, ts.WatchTaskPass)
	}
}

// ExecuteTask executes a task
// this function blocks until the task is executed or the context cancelled
func (ts *TaskScheduler) ExecuteTask(ctx context.Context, task types.Task, taskWatchFn func(task types.Task, ctx context.Context, cancelFn context.CancelFunc)) error {
	runningTask := &types.TaskStatus{
		IsRunning: true,
	}
	defer func() {
		runningTask.IsRunning = false
	}()

	// check if task has already been started/executed
	ts.runningMutex.Lock()
	if ts.runningTasks[task] != nil {
		ts.runningMutex.Unlock()
		return fmt.Errorf("task has already been executed")
	}
	ts.runningTasks[task] = runningTask
	ts.runningMutex.Unlock()

	// create cancelable task context
	var taskCtx context.Context
	var taskCancelFn context.CancelFunc
	taskTimeout := task.Timeout()
	if taskTimeout > 0 {
		taskCtx, taskCancelFn = context.WithTimeout(ctx, taskTimeout)
	} else {
		taskCtx, taskCancelFn = context.WithCancel(ctx)
	}
	defer taskCancelFn()

	defer func() {
		err := task.Cleanup(ctx)
		if err != nil {
			task.Logger().Errorf("task cleanup failed: %v", err)
		}
	}()

	// run task watcher if supplied
	if taskWatchFn != nil {
		go taskWatchFn(task, taskCtx, taskCancelFn)
	}

	// execute task
	task.Logger().Infof("starting task")
	err := task.Execute(taskCtx)
	if err != nil {
		task.Logger().Errorf("task failed: %v", err)
		runningTask.Error = err
		return fmt.Errorf("task failed: %w", err)
	}

	task.Logger().Infof("task completed", err)
	return nil
}

func (ts *TaskScheduler) WatchTaskPass(task types.Task, ctx context.Context, cancelFn context.CancelFunc) {
	// poll task result and cancel context when task result is passed or failed
	for {
		pollInterval := task.PollingInterval()
		if pollInterval == 0 {
			pollInterval = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
			taskResult, _ := task.GetResult()
			if taskResult == types.TaskResultSuccess || taskResult == types.TaskResultFailure {
				cancelFn()
				return
			}
		}
	}
}

func (ts *TaskScheduler) GetTaskStatus(task types.Task) *types.TaskStatus {
	ts.runningMutex.Lock()
	defer ts.runningMutex.Unlock()
	return ts.runningTasks[task]
}
