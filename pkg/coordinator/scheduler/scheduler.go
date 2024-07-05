package scheduler

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

type TaskScheduler struct {
	services         types.TaskServices
	logger           logrus.FieldLogger
	rootVars         types.Variables
	taskCount        types.TaskIndex
	allTasks         []types.TaskIndex
	rootTasks        []types.TaskIndex
	allCleanupTasks  []types.TaskIndex
	rootCleanupTasks []types.TaskIndex
	taskStateMutex   sync.RWMutex
	taskStateMap     map[types.TaskIndex]*taskState
	cancelTaskCtx    context.CancelFunc
	cancelCleanupCtx context.CancelFunc
}

func NewTaskScheduler(log logrus.FieldLogger, services types.TaskServices, variables types.Variables) *TaskScheduler {
	return &TaskScheduler{
		logger:       log,
		rootVars:     variables,
		taskCount:    1,
		rootTasks:    make([]types.TaskIndex, 0),
		allTasks:     make([]types.TaskIndex, 0),
		taskStateMap: make(map[types.TaskIndex]*taskState),
		services:     services,
	}
}

func (ts *TaskScheduler) GetServices() types.TaskServices {
	return ts.services
}

func (ts *TaskScheduler) AddRootTask(options *types.TaskOptions) (types.TaskIndex, error) {
	task, err := ts.newTask(options, nil, nil, false)
	if err != nil {
		return 0, err
	}

	ts.rootTasks = append(ts.rootTasks, task.index)

	return task.index, nil
}

func (ts *TaskScheduler) AddCleanupTask(options *types.TaskOptions) (types.TaskIndex, error) {
	task, err := ts.newTask(options, nil, nil, true)
	if err != nil {
		return 0, err
	}

	ts.rootCleanupTasks = append(ts.rootCleanupTasks, task.index)

	return task.index, nil
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

func (ts *TaskScheduler) CancelTasks(cancelCleanup bool) {
	if ts.cancelTaskCtx != nil {
		ts.cancelTaskCtx()

		if cancelCleanup {
			ts.cancelCleanupCtx()
		}
	}
}

func (ts *TaskScheduler) getTaskState(taskIndex types.TaskIndex) *taskState {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	return ts.taskStateMap[taskIndex]
}

func (ts *TaskScheduler) GetTaskState(taskIndex types.TaskIndex) types.TaskState {
	return ts.getTaskState(taskIndex)
}

func (ts *TaskScheduler) GetAllTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.allTasks))
	copy(taskList, ts.allTasks)
	ts.sortTaskList(taskList)

	return taskList
}

func (ts *TaskScheduler) GetTaskCount() int {
	if ts == nil {
		return 0
	}

	return len(ts.allTasks)
}

func (ts *TaskScheduler) GetRootTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.rootTasks))
	copy(taskList, ts.rootTasks)

	return taskList
}

func (ts *TaskScheduler) GetAllCleanupTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.allCleanupTasks))
	copy(taskList, ts.allCleanupTasks)
	ts.sortTaskList(taskList)

	return taskList
}

func (ts *TaskScheduler) GetRootCleanupTasks() []types.TaskIndex {
	ts.taskStateMutex.RLock()
	defer ts.taskStateMutex.RUnlock()

	taskList := make([]types.TaskIndex, len(ts.rootCleanupTasks))
	copy(taskList, ts.rootCleanupTasks)

	return taskList
}

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
