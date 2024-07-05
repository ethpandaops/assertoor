package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/tasks"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
)

type taskState struct {
	index       types.TaskIndex
	options     *types.TaskOptions
	descriptor  *types.TaskDescriptor
	task        types.Task
	taskDepth   uint64
	taskVars    types.Variables
	logger      *logger.LogScope
	parentState *taskState

	isCleanup bool
	isStarted bool
	isRunning bool
	isTimeout bool
	startTime time.Time
	stopTime  time.Time

	taskConfig     interface{}
	taskOutputs    types.Variables
	taskStatusVars types.Variables

	updatedResult    bool
	taskResult       types.TaskResult
	taskError        error
	resultNotifyChan chan bool
	resultMutex      sync.RWMutex
}

func (ts *TaskScheduler) newTaskState(options *types.TaskOptions, parentState *taskState, variables types.Variables, isCleanupTask bool) (*taskState, error) {
	if variables == nil {
		if parentState != nil {
			variables = parentState.taskVars
		} else {
			variables = ts.rootVars
		}
	}

	// lookup task descriptor by name
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

	// create task state
	taskIdx := ts.taskCount
	taskState := &taskState{
		index:       taskIdx,
		options:     options,
		descriptor:  taskDescriptor,
		parentState: parentState,
		taskVars:    variables,
		logger: logger.NewLogger(&logger.ScopeOptions{
			Parent:      ts.logger.WithField("task", options.Name).WithField("taskidx", taskIdx),
			HistorySize: 1000,
		}),
		taskOutputs:    vars.NewVariables(nil),
		taskStatusVars: vars.NewVariables(nil),
	}

	if parentState != nil {
		taskState.parentState = parentState
		taskState.taskDepth = parentState.taskDepth + 1
	}

	taskState.taskStatusVars.NewSubScope("outputs")

	if options.ID != "" {
		tasksScope := variables.GetSubScope("tasks")
		tasksScope.SetSubScope(options.ID, taskState.taskStatusVars)
	}

	ts.taskCount++

	// create internal execution state
	ts.taskStateMutex.Lock()
	ts.taskStateMap[taskIdx] = taskState

	if isCleanupTask {
		ts.allCleanupTasks = append(ts.allCleanupTasks, taskIdx)
	} else {
		ts.allTasks = append(ts.allTasks, taskIdx)
	}
	ts.taskStateMutex.Unlock()

	return taskState, nil
}

func (ts *taskState) setTaskResult(result types.TaskResult, setUpdated bool) {
	ts.resultMutex.Lock()
	defer ts.resultMutex.Unlock()

	if setUpdated {
		ts.updatedResult = true
	}

	if ts.taskResult == result {
		return
	}

	ts.taskResult = result
	ts.taskStatusVars.SetVar("result", uint8(result))

	if ts.resultNotifyChan != nil {
		close(ts.resultNotifyChan)
		ts.resultNotifyChan = nil
	}
}

func (ts *taskState) GetTaskStatus() *types.TaskStatus {
	taskStatus := &types.TaskStatus{
		Index:       ts.index,
		ParentIndex: 0,
		IsStarted:   ts.isStarted,
		IsRunning:   ts.isRunning,
		StartTime:   ts.startTime,
		StopTime:    ts.stopTime,
		Result:      ts.taskResult,
		Error:       ts.taskError,
		Logger:      ts.logger,
	}
	if ts.parentState != nil {
		taskStatus.ParentIndex = ts.parentState.index
	}

	return taskStatus
}

func (ts *taskState) GetTaskStatusVars() types.Variables {
	return ts.taskStatusVars
}

func (ts *taskState) GetTaskResultUpdateChan(oldResult types.TaskResult) <-chan bool {
	ts.resultMutex.RLock()
	defer ts.resultMutex.RUnlock()

	if ts.taskResult != oldResult {
		return nil
	}

	if ts.resultNotifyChan == nil {
		ts.resultNotifyChan = make(chan bool)
	}

	return ts.resultNotifyChan
}

func (ts *taskState) Index() types.TaskIndex {
	return ts.index
}

func (ts *taskState) ParentIndex() types.TaskIndex {
	if ts.parentState != nil {
		return ts.parentState.index
	}

	return 0
}

func (ts *taskState) ID() string {
	return ts.options.ID
}

func (ts *taskState) Name() string {
	return ts.options.Name
}

func (ts *taskState) Title() string {
	return ts.taskVars.ResolvePlaceholders(ts.options.Title)
}

func (ts *taskState) Description() string {
	return ts.descriptor.Description
}

func (ts *taskState) Config() interface{} {
	if ts.task != nil {
		return ts.task.Config()
	}

	return ts.taskConfig
}

func (ts *taskState) Timeout() time.Duration {
	if ts.task != nil {
		return ts.task.Timeout()
	}

	return ts.options.Timeout.Duration
}
