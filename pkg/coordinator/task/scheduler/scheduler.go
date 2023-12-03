package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
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
	taskStateMutex   sync.RWMutex
	taskStateMap     map[types.Task]*taskExecutionState

	clientPool *clients.ClientPool
}

type taskExecutionState struct {
	isStarted        bool
	isRunning        bool
	isTimeout        bool
	updatedResult    bool
	taskResult       types.TaskResult
	taskError        error
	resultNotifyChan chan bool
	resultMutex      sync.RWMutex
}

func NewTaskScheduler(logger logrus.FieldLogger, clientPool *clients.ClientPool) *TaskScheduler {
	return &TaskScheduler{
		logger:       logger,
		rootTasks:    make([]types.Task, 0),
		allTasks:     make([]types.Task, 0),
		taskStateMap: make(map[types.Task]*taskExecutionState),
		clientPool:   clientPool,
	}
}

func (ts *TaskScheduler) GetLogger() logrus.FieldLogger {
	return ts.logger
}

func (ts *TaskScheduler) GetTaskCount() int {
	return len(ts.allTasks)
}

func (ts *TaskScheduler) GetClientPool() *clients.ClientPool {
	return ts.clientPool
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
		ParentTask: parent,
		NewTask: func(options *types.TaskOptions) (types.Task, error) {
			return ts.newTask(options, task, isCleanupTask)
		},
		SetResult: func(result types.TaskResult) {
			ts.setTaskResult(task, result, true)
		},
	}
	ts.taskCount++
	var err error
	task, err = taskDescriptor.NewTask(taskCtx, options)
	if err != nil {
		return nil, fmt.Errorf("failed task '%v' initialization: %w", options.Name, err)
	}

	// create internal execution state
	ts.taskStateMutex.Lock()
	taskState := &taskExecutionState{}
	ts.taskStateMap[task] = taskState
	ts.taskStateMutex.Unlock()

	if isCleanupTask {
		ts.allCleanupTasks = append(ts.allCleanupTasks, task)
	} else {
		ts.allTasks = append(ts.allTasks, task)
	}
	return task, nil
}

func (ts *TaskScheduler) setTaskResult(task types.Task, result types.TaskResult, setUpdated bool) {
	ts.taskStateMutex.RLock()
	taskState := ts.taskStateMap[task]
	ts.taskStateMutex.RUnlock()
	if taskState == nil {
		return
	}

	taskState.resultMutex.Lock()
	defer taskState.resultMutex.Unlock()
	if setUpdated {
		taskState.updatedResult = true
	}
	if taskState.taskResult == result {
		return
	}
	taskState.taskResult = result
	if taskState.resultNotifyChan != nil {
		close(taskState.resultNotifyChan)
		taskState.resultNotifyChan = nil
	}
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
	// check if task has already been started/executed
	ts.taskStateMutex.RLock()
	taskState := ts.taskStateMap[task]
	ts.taskStateMutex.RUnlock()
	if taskState == nil {
		return fmt.Errorf("task state not found")
	}
	if taskState.isStarted {
		return fmt.Errorf("task has already been executed")
	}
	taskState.isStarted = true
	taskState.isRunning = true
	defer func() {
		taskState.isRunning = false
	}()

	// create cancelable task context
	taskCtx, taskCancelFn := context.WithCancel(ctx)
	taskTimeout := task.Timeout()
	if taskTimeout > 0 {
		go func() {
			select {
			case <-time.After(taskTimeout):
				task.Logger().Warnf("task timed out")
				taskState.isTimeout = true
				taskCancelFn()
			case <-taskCtx.Done():
			}
		}()
	}
	defer taskCancelFn()

	defer func() {
		if r := recover(); r != nil {
			err := r.(error)
			task.Logger().Errorf("task execution panic: %v", r)
			taskState.taskError = err
			ts.setTaskResult(task, types.TaskResultFailure, false)
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
		task.Logger().Errorf("task execution returned error: %v", err)
		if taskState.taskError == nil {
			taskState.taskError = err
		}
	}

	// set task result
	if !taskState.updatedResult || taskState.taskResult == types.TaskResultNone {
		// set task result if not already done by task
		if taskState.isTimeout || err != nil {
			ts.setTaskResult(task, types.TaskResultFailure, false)
		} else {
			ts.setTaskResult(task, types.TaskResultSuccess, false)
		}
	}

	if err != nil {
		return fmt.Errorf("task failed: %w", err)
	}
	if taskState.taskResult == types.TaskResultFailure {
		task.Logger().Warnf("task failed with failure result")
		return fmt.Errorf("task failed")
	}
	task.Logger().Infof("task completed")
	return nil
}

func (ts *TaskScheduler) WatchTaskPass(task types.Task, ctx context.Context, cancelFn context.CancelFunc) {
	// poll task result and cancel context when task result is passed or failed
	for {
		updateChan := ts.GetTaskResultUpdateChan(task, types.TaskResultNone)
		if updateChan != nil {
			select {
			case <-ctx.Done():
				return
			case <-updateChan:
			}
		}
		taskStatus := ts.GetTaskStatus(task)
		if taskStatus.Result != types.TaskResultNone {
			cancelFn()
			return
		}
	}
}

func (ts *TaskScheduler) GetTaskStatus(task types.Task) *types.TaskStatus {
	ts.taskStateMutex.RLock()
	taskState := ts.taskStateMap[task]
	ts.taskStateMutex.RUnlock()
	if taskState == nil {
		return nil
	}
	return &types.TaskStatus{
		IsStarted: taskState.isStarted,
		IsRunning: taskState.isRunning,
		Result:    taskState.taskResult,
		Error:     taskState.taskError,
	}
}

func (ts *TaskScheduler) GetTaskResultUpdateChan(task types.Task, oldResult types.TaskResult) <-chan bool {
	ts.taskStateMutex.RLock()
	taskState := ts.taskStateMap[task]
	ts.taskStateMutex.RUnlock()
	if taskState == nil {
		return nil
	}
	taskState.resultMutex.RLock()
	defer taskState.resultMutex.RUnlock()
	if taskState.taskResult != oldResult {
		return nil
	}
	if taskState.resultNotifyChan == nil {
		taskState.resultNotifyChan = make(chan bool)
	}
	return taskState.resultNotifyChan
}
