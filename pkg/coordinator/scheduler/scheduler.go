package scheduler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

// TaskScheduler is a struct that manages the execution of tasks.
type TaskScheduler struct {
	services         types.TaskServices
	logger           logrus.FieldLogger
	rootVars         types.Variables
	nextTaskIndex    types.TaskIndex
	allTasks         []types.TaskIndex
	rootTasks        []types.TaskIndex
	allCleanupTasks  []types.TaskIndex
	rootCleanupTasks []types.TaskIndex
	taskStateMutex   sync.RWMutex
	taskStateMap     map[types.TaskIndex]*taskState
	cancelTaskCtx    context.CancelFunc
	cancelCleanupCtx context.CancelFunc
}

// NewTaskScheduler creates a new TaskScheduler instance.
func NewTaskScheduler(log logrus.FieldLogger, services types.TaskServices, variables types.Variables) *TaskScheduler {
	return &TaskScheduler{
		logger:        log,
		rootVars:      variables,
		nextTaskIndex: 1,
		rootTasks:     make([]types.TaskIndex, 0),
		allTasks:      make([]types.TaskIndex, 0),
		taskStateMap:  make(map[types.TaskIndex]*taskState),
		services:      services,
	}
}

// GetServices returns the task services associated with the scheduler.
func (ts *TaskScheduler) GetServices() types.TaskServices {
	return ts.services
}

// ParseTaskOptions parses the raw task options and returns the parsed task options.
func (ts *TaskScheduler) ParseTaskOptions(rawtask *helper.RawMessage) (*types.TaskOptions, error) {
	options := &types.TaskOptions{}
	if err := rawtask.Unmarshal(&options); err != nil {
		return nil, fmt.Errorf("error parsing task: %w", err)
	}

	return options, nil
}

// AddRootTask adds a root task to the scheduler and returns the task index.
func (ts *TaskScheduler) AddRootTask(options *types.TaskOptions) (types.TaskIndex, error) {
	task, err := ts.newTaskState(options, nil, nil, false)
	if err != nil {
		return 0, err
	}

	ts.rootTasks = append(ts.rootTasks, task.index)

	return task.index, nil
}

// AddCleanupTask adds a cleanup task to the scheduler and returns the task index.
func (ts *TaskScheduler) AddCleanupTask(options *types.TaskOptions) (types.TaskIndex, error) {
	task, err := ts.newTaskState(options, nil, nil, true)
	if err != nil {
		return 0, err
	}

	ts.rootCleanupTasks = append(ts.rootCleanupTasks, task.index)

	return task.index, nil
}

// RunTasks runs all the tasks in the scheduler within the specified timeout.
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

// runCleanupTasks runs all the cleanup tasks in the scheduler.
func (ts *TaskScheduler) runCleanupTasks(ctx context.Context) {
	for _, taskIndex := range ts.rootCleanupTasks {
		if ctx.Err() != nil {
			return
		}

		err := ts.ExecuteTask(ctx, taskIndex, ts.WatchTaskPass)
		if err != nil {
			taskState := ts.getTaskState(taskIndex)
			taskState.logger.GetLogger().Errorf("cleanup task failed: %v", err)
		}
	}
}

// CancelTasks cancels all the tasks in the scheduler.
func (ts *TaskScheduler) CancelTasks(cancelCleanup bool) {
	if ts.cancelTaskCtx != nil {
		ts.cancelTaskCtx()

		if cancelCleanup {
			ts.cancelCleanupCtx()
		}
	}
}

// getTaskState returns the internal taskState of the task with the given task index.
func (ts *TaskScheduler) getTaskState(taskIndex types.TaskIndex) *taskState {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	return ts.taskStateMap[taskIndex]
}

// GetTaskState returns the TaskState interface of the task with the given task index.
func (ts *TaskScheduler) GetTaskState(taskIndex types.TaskIndex) types.TaskState {
	return ts.getTaskState(taskIndex)
}

// GetAllTasks returns a list of all task indices in the scheduler.
func (ts *TaskScheduler) GetAllTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.allTasks))
	copy(taskList, ts.allTasks)
	ts.sortTaskList(taskList)

	return taskList
}

// GetTaskCount returns the total number of tasks in the scheduler.
func (ts *TaskScheduler) GetTaskCount() int {
	if ts == nil {
		return 0
	}

	return len(ts.allTasks)
}

// GetRootTasks returns a list of root task indices in the scheduler.
func (ts *TaskScheduler) GetRootTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.rootTasks))
	copy(taskList, ts.rootTasks)

	return taskList
}

// GetAllCleanupTasks returns a list of all cleanup task indices in the scheduler.
func (ts *TaskScheduler) GetAllCleanupTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.allCleanupTasks))
	copy(taskList, ts.allCleanupTasks)
	ts.sortTaskList(taskList)

	return taskList
}

// GetRootCleanupTasks returns a list of root cleanup task indices in the scheduler.
func (ts *TaskScheduler) GetRootCleanupTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.rootCleanupTasks))
	copy(taskList, ts.rootCleanupTasks)

	return taskList
}

// sortTaskList sorts the task list based on the task relationships and indices.
func (ts *TaskScheduler) sortTaskList(taskList []types.TaskIndex) {
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
