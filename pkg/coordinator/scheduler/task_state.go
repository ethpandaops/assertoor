package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/logger"
	"github.com/erigontech/assertoor/pkg/coordinator/tasks"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	"github.com/jmoiron/sqlx"
	"gopkg.in/yaml.v3"
)

type taskState struct {
	ts          *TaskScheduler
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
	isSkipped bool
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

	dbTaskState *db.TaskState
}

func (ts *TaskScheduler) newTaskState(options *types.TaskOptions, parentState *taskState, variables types.Variables, isCleanupTask bool) (*taskState, error) {
	if variables == nil {
		if parentState != nil {
			variables = parentState.taskVars
		} else {
			variables = ts.rootVars
		}
	}

	taskDescriptor := tasks.GetTaskDescriptor(options.Name)
	if taskDescriptor == nil {
		return nil, fmt.Errorf("unknown task name: %v", options.Name)
	}

	// create task state
	ts.taskStateMutex.Lock()
	defer ts.taskStateMutex.Unlock()

	ts.taskCount++
	taskIdx := ts.taskCount
	taskState := &taskState{
		ts:          ts,
		index:       taskIdx,
		options:     options,
		descriptor:  taskDescriptor,
		parentState: parentState,
		taskVars:    variables,
		isCleanup:   isCleanupTask,
		logger: logger.NewLogger(&logger.ScopeOptions{
			Parent:     ts.logger.WithField("task", options.Name).WithField("taskidx", taskIdx),
			BufferSize: 1000,
			Database:   ts.services.Database(),
			TestRunID:  ts.testRunID,
			TaskID:     uint64(taskIdx),
		}),
		taskOutputs:    vars.NewVariables(nil),
		taskStatusVars: vars.NewVariables(nil),
	}

	if parentState != nil {
		taskState.parentState = parentState
		taskState.taskDepth = parentState.taskDepth + 1
	}

	taskState.taskStatusVars.SetSubScope("outputs", taskState.taskOutputs)

	if options.ID != "" {
		tasksScope := variables.GetSubScope("tasks")
		tasksScope.SetSubScope(options.ID, taskState.taskStatusVars)
	}

	ts.taskStateMap[taskIdx] = taskState

	if isCleanupTask {
		ts.allCleanupTasks = append(ts.allCleanupTasks, taskIdx)
	} else {
		ts.allTasks = append(ts.allTasks, taskIdx)
	}

	// add to database
	if database := ts.services.Database(); database != nil {
		taskState.dbTaskState = &db.TaskState{
			RunID:   ts.testRunID,
			TaskID:  uint64(taskIdx),
			Name:    taskState.options.Name,
			Title:   taskState.Title(),
			RefID:   taskState.options.ID,
			Timeout: int64(taskState.options.Timeout.Seconds()),
			IfCond:  taskState.options.If,
		}

		if taskState.isCleanup {
			taskState.dbTaskState.RunFlags |= db.TaskRunFlagCleanup
		}

		if parentState != nil {
			taskState.dbTaskState.ParentTask = uint64(parentState.index)
		}

		err := database.RunTransaction(func(tx *sqlx.Tx) error {
			return database.InsertTaskState(tx, taskState.dbTaskState)
		})
		if err != nil {
			return nil, err
		}
	}

	return taskState, nil
}

func (ts *taskState) updateTaskState() error {
	if ts.dbTaskState == nil {
		return nil
	}

	changedFields := []string{}

	if ts.Title() != ts.dbTaskState.Title {
		ts.dbTaskState.Title = ts.Title()

		changedFields = append(changedFields, "title")
	}

	runFlags := uint32(0)

	if ts.isCleanup {
		runFlags |= db.TaskRunFlagCleanup
	}

	if ts.isStarted {
		runFlags |= db.TaskRunFlagStarted
	}

	if ts.isRunning {
		runFlags |= db.TaskRunFlagRunning
	}

	if ts.isSkipped {
		runFlags |= db.TaskRunFlagSkipped
	}

	if ts.isTimeout {
		runFlags |= db.TaskRunFlagTimeout
	}

	if runFlags != ts.dbTaskState.RunFlags {
		ts.dbTaskState.RunFlags = runFlags

		changedFields = append(changedFields, "run_flags")
	}

	if !ts.startTime.IsZero() && ts.startTime.UnixMilli() != ts.dbTaskState.StartTime {
		ts.dbTaskState.StartTime = ts.startTime.UnixMilli()

		changedFields = append(changedFields, "start_time")
	}

	if !ts.stopTime.IsZero() && ts.stopTime.UnixMilli() != ts.dbTaskState.StopTime {
		ts.dbTaskState.StopTime = ts.stopTime.UnixMilli()

		changedFields = append(changedFields, "stop_time")
	}

	taskStatusVars := ts.taskStatusVars.GetVarsMap(nil, false)

	configVarsYaml, err := yaml.Marshal(ts.Config())
	if err != nil {
		return err
	}

	if string(configVarsYaml) != ts.dbTaskState.TaskConfig {
		ts.dbTaskState.TaskConfig = string(configVarsYaml)

		changedFields = append(changedFields, "task_config")
	}

	statusVarsYaml, err := yaml.Marshal(taskStatusVars)
	if err != nil {
		return err
	}

	if string(statusVarsYaml) != ts.dbTaskState.TaskStatus {
		ts.dbTaskState.TaskStatus = string(statusVarsYaml)

		changedFields = append(changedFields, "task_status")
	}

	if int(ts.taskResult) != ts.dbTaskState.TaskResult {
		ts.dbTaskState.TaskResult = int(ts.taskResult)

		changedFields = append(changedFields, "task_result")
	}

	if ts.taskError != nil && ts.taskError.Error() != ts.dbTaskState.TaskError {
		ts.dbTaskState.TaskError = ts.taskError.Error()

		changedFields = append(changedFields, "task_error")
	}

	if len(changedFields) == 0 {
		return nil
	}

	if database := ts.ts.services.Database(); database != nil {
		return database.RunTransaction(func(tx *sqlx.Tx) error {
			return database.UpdateTaskStateStatus(tx, ts.dbTaskState, changedFields)
		})
	}

	return nil
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

	if err := ts.updateTaskState(); err != nil {
		ts.logger.GetLogger().Errorf("failed to update task state in db: %v", err)
	}

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
		IsSkipped:   ts.isSkipped,
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

func (ts *taskState) GetScopeOwner() types.TaskIndex {
	scopeOwner, found := ts.taskVars.LookupVar("scopeOwner")
	if !found {
		return 0
	}

	if scopeOwnerInt, ok := scopeOwner.(uint64); ok {
		return types.TaskIndex(scopeOwnerInt)
	}

	return 0
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
