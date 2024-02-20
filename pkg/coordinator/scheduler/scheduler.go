package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"sort"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/tasks"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

type TaskScheduler struct {
	services         types.TaskServices
	logger           logrus.FieldLogger
	rootVars         types.Variables
	taskCount        uint64
	allTasks         []types.Task
	rootTasks        []types.Task
	allCleanupTasks  []types.Task
	rootCleanupTasks []types.Task
	taskStateMutex   sync.RWMutex
	taskStateMap     map[types.Task]*taskExecutionState
	cancelTaskCtx    context.CancelFunc
	cancelCleanupCtx context.CancelFunc
}

type taskExecutionState struct {
	index            uint64
	task             types.Task
	taskDepth        uint64
	taskVars         types.Variables
	logger           *logger.LogScope
	parentState      *taskExecutionState
	isStarted        bool
	isRunning        bool
	isTimeout        bool
	startTime        time.Time
	stopTime         time.Time
	updatedResult    bool
	taskResult       types.TaskResult
	taskError        error
	resultNotifyChan chan bool
	resultMutex      sync.RWMutex
}

func NewTaskScheduler(log logrus.FieldLogger, services types.TaskServices, variables types.Variables) *TaskScheduler {
	return &TaskScheduler{
		logger:       log,
		rootVars:     variables,
		taskCount:    1,
		rootTasks:    make([]types.Task, 0),
		allTasks:     make([]types.Task, 0),
		taskStateMap: make(map[types.Task]*taskExecutionState),
		services:     services,
	}
}

func (ts *TaskScheduler) GetTaskCount() int {
	if ts == nil {
		return 0
	}

	return len(ts.allTasks)
}

func (ts *TaskScheduler) GetServices() types.TaskServices {
	return ts.services
}

func (ts *TaskScheduler) AddRootTask(options *types.TaskOptions) (types.Task, error) {
	task, err := ts.newTask(options, nil, nil, false)
	if err != nil {
		return nil, err
	}

	ts.rootTasks = append(ts.rootTasks, task)

	return task, nil
}

func (ts *TaskScheduler) AddCleanupTask(options *types.TaskOptions) (types.Task, error) {
	task, err := ts.newTask(options, nil, nil, true)
	if err != nil {
		return nil, err
	}

	ts.rootCleanupTasks = append(ts.rootCleanupTasks, task)

	return task, nil
}

func (ts *TaskScheduler) newTask(options *types.TaskOptions, parentState *taskExecutionState, variables types.Variables, isCleanupTask bool) (types.Task, error) {
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

	if variables == nil {
		if parentState != nil {
			variables = parentState.taskVars
		} else {
			variables = ts.rootVars
		}
	}

	// create instance of task
	var task types.Task

	taskIdx := ts.taskCount
	taskState := &taskExecutionState{
		index:       taskIdx,
		parentState: parentState,
		taskVars:    variables,
		logger: logger.NewLogger(&logger.ScopeOptions{
			Parent:      ts.logger.WithField("task", taskDescriptor.Name).WithField("taskidx", taskIdx),
			HistorySize: 1000,
		}),
	}

	if parentState != nil {
		taskState.parentState = parentState
		taskState.taskDepth = parentState.taskDepth + 1
	}

	taskCtx := &types.TaskContext{
		Scheduler: ts,
		Index:     taskIdx,
		Vars:      variables,
		Logger:    taskState.logger,
		NewTask: func(options *types.TaskOptions, variables types.Variables) (types.Task, error) {
			return ts.newTask(options, taskState, variables, isCleanupTask)
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
	taskState.task = task
	ts.taskStateMap[task] = taskState

	if isCleanupTask {
		ts.allCleanupTasks = append(ts.allCleanupTasks, task)
	} else {
		ts.allTasks = append(ts.allTasks, task)
	}
	ts.taskStateMutex.Unlock()

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

func (ts *TaskScheduler) RunTasks(ctx context.Context, timeout time.Duration) error {
	var cleanupCtx, tasksCtx context.Context

	cleanupCtx, ts.cancelCleanupCtx = context.WithCancel(ctx)

	defer ts.runCleanupTasks(cleanupCtx)

	if timeout > 0 {
		tasksCtx, ts.cancelTaskCtx = context.WithTimeout(ctx, timeout)
	} else {
		tasksCtx, ts.cancelTaskCtx = context.WithCancel(ctx)
	}

	defer ts.cancelTaskCtx()

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

		err := ts.ExecuteTask(ctx, task, ts.WatchTaskPass)
		if err != nil {
			task.Logger().Errorf("cleanup task failed: %v", err)
		}
	}
}

// ExecuteTask executes a task
// this function blocks until the task is executed or the context cancelled
func (ts *TaskScheduler) ExecuteTask(ctx context.Context, task types.Task, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, task types.Task)) error {
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
	taskState.startTime = time.Now()
	taskState.isRunning = true

	defer func() {
		taskState.isRunning = false
		taskState.stopTime = time.Now()
	}()

	// load task config
	err := task.LoadConfig()
	if err != nil {
		task.Logger().Errorf("config validation failed: %v", err)
		ts.setTaskResult(task, types.TaskResultFailure, false)

		return fmt.Errorf("task %v config validation failed: %w", task.Name(), err)
	}

	// create cancelable task context
	taskCtx, taskCancelFn := context.WithCancel(ctx)
	taskTimeout := task.Timeout()

	if taskTimeout > 0 {
		go func() {
			select {
			case <-time.After(taskTimeout):
				taskState.isTimeout = true

				task.Logger().Warnf("task timed out")
				taskCancelFn()
			case <-taskCtx.Done():
			}
		}()
	}

	defer taskCancelFn()

	defer func() {
		if r := recover(); r != nil {
			pErr, ok := r.(error)
			if ok {
				taskState.taskError = pErr
			}

			task.Logger().Errorf("task execution panic: %v, stack: %v", r, string(debug.Stack()))
			ts.setTaskResult(task, types.TaskResultFailure, false)
		}
	}()

	// run task watcher if supplied
	if taskWatchFn != nil {
		go taskWatchFn(taskCtx, taskCancelFn, task)
	}

	// execute task
	task.Logger().Infof("starting task")
	err = task.Execute(taskCtx)

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

	if taskState.taskResult == types.TaskResultFailure {
		task.Logger().Warnf("task failed with failure result: %v", taskState.taskError)
		return fmt.Errorf("task failed: %w", taskState.taskError)
	}

	task.Logger().Infof("task completed")

	return nil
}

func (ts *TaskScheduler) WatchTaskPass(ctx context.Context, cancelFn context.CancelFunc, task types.Task) {
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

func (ts *TaskScheduler) CancelTasks(cancelCleanup bool) {
	if ts.cancelTaskCtx != nil {
		ts.cancelTaskCtx()

		if cancelCleanup {
			ts.cancelCleanupCtx()
		}
	}
}

func (ts *TaskScheduler) GetAllTasks() []types.Task {
	ts.taskStateMutex.RLock()
	taskList := make([]types.Task, len(ts.allTasks))
	copy(taskList, ts.allTasks)
	ts.sortTaskList(taskList)
	ts.taskStateMutex.RUnlock()

	return taskList
}

func (ts *TaskScheduler) GetRootTasks() []types.Task {
	ts.taskStateMutex.RLock()
	taskList := make([]types.Task, len(ts.rootTasks))
	copy(taskList, ts.rootTasks)
	ts.taskStateMutex.RUnlock()

	return taskList
}

func (ts *TaskScheduler) GetAllCleanupTasks() []types.Task {
	ts.taskStateMutex.RLock()
	taskList := make([]types.Task, len(ts.allCleanupTasks))
	copy(taskList, ts.allCleanupTasks)
	ts.sortTaskList(taskList)
	ts.taskStateMutex.RUnlock()

	return taskList
}

func (ts *TaskScheduler) GetRootCleanupTasks() []types.Task {
	ts.taskStateMutex.RLock()
	taskList := make([]types.Task, len(ts.rootCleanupTasks))
	copy(taskList, ts.rootCleanupTasks)
	ts.taskStateMutex.RUnlock()

	return taskList
}

func (ts *TaskScheduler) sortTaskList(taskList []types.Task) {
	sort.Slice(taskList, func(a, b int) bool {
		taskStateA := ts.taskStateMap[taskList[a]]
		taskStateB := ts.taskStateMap[taskList[b]]

		if taskStateA.parentState == taskStateB.parentState {
			return taskStateA.index < taskStateB.index
		}

		for {
			switch {
			case taskStateA.parentState == taskStateB:
				return false
			case taskStateB.parentState == taskStateA:
				return true
			case taskStateA.taskDepth > taskStateB.taskDepth:
				taskStateA = taskStateA.parentState
			case taskStateB.taskDepth > taskStateA.taskDepth:
				taskStateB = taskStateB.parentState
			default:
				taskStateA = taskStateA.parentState
				taskStateB = taskStateB.parentState
			}

			if taskStateA.parentState == taskStateB.parentState {
				return taskStateA.index < taskStateB.index
			}
		}
	})
}

func (ts *TaskScheduler) GetTaskStatus(task types.Task) *types.TaskStatus {
	ts.taskStateMutex.RLock()
	taskState := ts.taskStateMap[task]
	ts.taskStateMutex.RUnlock()

	if taskState == nil {
		return nil
	}

	taskStatus := &types.TaskStatus{
		Index:       taskState.index,
		ParentIndex: 0,
		IsStarted:   taskState.isStarted,
		IsRunning:   taskState.isRunning,
		StartTime:   taskState.startTime,
		StopTime:    taskState.stopTime,
		Result:      taskState.taskResult,
		Error:       taskState.taskError,
		Logger:      taskState.logger,
	}
	if taskState.parentState != nil {
		taskStatus.ParentIndex = taskState.parentState.index
	}

	return taskStatus
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
