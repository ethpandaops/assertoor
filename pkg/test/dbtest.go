package test

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/logger"
	"github.com/ethpandaops/assertoor/pkg/tasks"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"gopkg.in/yaml.v3"
)

type dbTest struct {
	database *db.Database

	runID   uint64
	testRun *db.TestRun

	taskIndexMtx sync.Mutex
	taskIndex    []*db.TaskStateIndex
}

func LoadTestFromDB(database *db.Database, runID uint64) (types.Test, error) {
	// load test from database
	testRun, err := database.GetTestRunByRunID(runID)
	if err != nil {
		return nil, err
	}

	return &dbTest{
		database: database,
		runID:    runID,
		testRun:  testRun,
	}, nil
}

func WrapDBTestRun(database *db.Database, test *db.TestRun) types.Test {
	return &dbTest{
		database: database,
		runID:    test.RunID,
		testRun:  test,
	}
}

func (dbt *dbTest) RunID() uint64 {
	return dbt.runID
}

func (dbt *dbTest) TestID() string {
	return dbt.testRun.TestID
}

func (dbt *dbTest) Name() string {
	return dbt.testRun.Name
}

func (dbt *dbTest) StartTime() time.Time {
	return time.UnixMilli(dbt.testRun.StartTime)
}

func (dbt *dbTest) StopTime() time.Time {
	return time.UnixMilli(dbt.testRun.StopTime)
}

func (dbt *dbTest) Timeout() time.Duration {
	return time.Duration(dbt.testRun.Timeout) * time.Second
}

func (dbt *dbTest) Status() types.TestStatus {
	return types.TestStatus(dbt.testRun.Status)
}

func (dbt *dbTest) GetTaskScheduler() types.TaskScheduler {
	return dbt
}

func (dbt *dbTest) AbortTest(_ bool) {}

func (dbt *dbTest) GetTaskCount() uint64 {
	dbt.loadTaskIndex()
	return uint64(len(dbt.taskIndex))
}

func (dbt *dbTest) loadTaskIndex() {
	dbt.taskIndexMtx.Lock()
	defer dbt.taskIndexMtx.Unlock()

	if dbt.taskIndex != nil {
		return
	}

	taskIndex, err := dbt.database.GetTaskStateIndex(dbt.runID)
	if err != nil {
		return
	}

	dbt.taskIndex = taskIndex
}

func (dbt *dbTest) GetAllTasks() []types.TaskIndex {
	dbt.loadTaskIndex()

	taskIDs := make([]types.TaskIndex, 0)

	for _, task := range dbt.taskIndex {
		if task.RunFlags&db.TaskRunFlagCleanup != 0 {
			continue
		}

		taskIDs = append(taskIDs, types.TaskIndex(task.TaskID))
	}

	dbt.sortTaskList(taskIDs)

	return taskIDs
}

func (dbt *dbTest) GetRootTasks() []types.TaskIndex {
	dbt.loadTaskIndex()

	taskIDs := make([]types.TaskIndex, 0)

	for _, task := range dbt.taskIndex {
		if task.ParentTask != 0 {
			continue
		}

		if task.RunFlags&db.TaskRunFlagCleanup != 0 {
			continue
		}

		taskIDs = append(taskIDs, types.TaskIndex(task.TaskID))
	}

	return taskIDs
}

func (dbt *dbTest) GetAllCleanupTasks() []types.TaskIndex {
	dbt.loadTaskIndex()

	taskIDs := make([]types.TaskIndex, 0)

	for _, task := range dbt.taskIndex {
		if task.RunFlags&db.TaskRunFlagCleanup == 0 {
			continue
		}

		taskIDs = append(taskIDs, types.TaskIndex(task.TaskID))
	}

	dbt.sortTaskList(taskIDs)

	return taskIDs
}

func (dbt *dbTest) GetRootCleanupTasks() []types.TaskIndex {
	dbt.loadTaskIndex()

	taskIDs := make([]types.TaskIndex, 0)

	for _, task := range dbt.taskIndex {
		if task.ParentTask != 0 {
			continue
		}

		if task.RunFlags&db.TaskRunFlagCleanup == 0 {
			continue
		}

		taskIDs = append(taskIDs, types.TaskIndex(task.TaskID))
	}

	return taskIDs
}

func (dbt *dbTest) sortTaskList(taskList []types.TaskIndex) {
	taskMap := map[types.TaskIndex]*db.TaskStateIndex{}

	for _, task := range dbt.taskIndex {
		taskMap[types.TaskIndex(task.TaskID)] = task
	}

	sort.Slice(taskList, func(a, b int) bool {
		taskStateA := taskMap[taskList[a]]
		taskStateB := taskMap[taskList[b]]

		if taskStateA == nil || taskStateB == nil {
			return false
		}

		if taskStateA.ParentTask == taskStateB.ParentTask {
			return taskStateA.TaskID < taskStateB.TaskID
		}

		for {
			switch {
			case taskStateA.ParentTask == uint64(taskList[b]):
				return false
			case taskStateB.ParentTask == uint64(taskList[a]):
				return true
			}

			taskStateADepth := 0
			for taskState := taskStateA; taskState != nil && taskState.ParentTask != 0; taskState = taskMap[types.TaskIndex(taskState.ParentTask)] {
				taskStateADepth++
			}

			taskStateBDepth := 0
			for taskState := taskStateB; taskState != nil && taskState.ParentTask != 0; taskState = taskMap[types.TaskIndex(taskState.ParentTask)] {
				taskStateBDepth++
			}

			switch {
			case taskStateADepth > taskStateBDepth:
				taskStateA = taskMap[types.TaskIndex(taskStateA.ParentTask)]
			case taskStateBDepth > taskStateADepth:
				taskStateB = taskMap[types.TaskIndex(taskStateB.ParentTask)]
			default:
				taskStateA = taskMap[types.TaskIndex(taskStateA.ParentTask)]
				taskStateB = taskMap[types.TaskIndex(taskStateB.ParentTask)]
			}

			if taskStateA.ParentTask == taskStateB.ParentTask {
				return taskStateA.TaskID < taskStateB.TaskID
			}
		}
	})
}

func (dbt *dbTest) GetTaskState(taskIndex types.TaskIndex) types.TaskState {
	task, err := dbt.database.GetTaskStateByTaskID(dbt.runID, uint64(taskIndex))
	if err != nil {
		return nil
	}

	return &dbTestTask{
		database:  dbt.database,
		taskState: task,
	}
}

type dbTestTask struct {
	database  *db.Database
	taskState *db.TaskState
}

func (dtt *dbTestTask) Index() types.TaskIndex {
	return types.TaskIndex(dtt.taskState.TaskID)
}

func (dtt *dbTestTask) ParentIndex() types.TaskIndex {
	return types.TaskIndex(dtt.taskState.ParentTask)
}

func (dtt *dbTestTask) ID() string {
	return dtt.taskState.RefID
}

func (dtt *dbTestTask) Name() string {
	return dtt.taskState.Name
}

func (dtt *dbTestTask) Title() string {
	return dtt.taskState.Title
}

func (dtt *dbTestTask) Description() string {
	taskDescriptor := tasks.GetTaskDescriptor(dtt.taskState.Name)
	if taskDescriptor != nil {
		return taskDescriptor.Description
	}

	return ""
}

func (dtt *dbTestTask) Config() interface{} {
	var config interface{}

	err := yaml.Unmarshal([]byte(dtt.taskState.TaskConfig), &config)
	if err != nil {
		return nil
	}

	return config
}

func (dtt *dbTestTask) Timeout() time.Duration {
	return time.Duration(dtt.taskState.Timeout) * time.Second
}

func (dtt *dbTestTask) GetTaskStatus() *types.TaskStatus {
	status := &types.TaskStatus{
		Index:       types.TaskIndex(dtt.taskState.TaskID),
		ParentIndex: types.TaskIndex(dtt.taskState.ParentTask),
		IsStarted:   dtt.taskState.RunFlags&db.TaskRunFlagStarted != 0,
		IsRunning:   dtt.taskState.RunFlags&db.TaskRunFlagRunning != 0,
		IsSkipped:   dtt.taskState.RunFlags&db.TaskRunFlagSkipped != 0,
		StartTime:   time.UnixMilli(dtt.taskState.StartTime),
		StopTime:    time.UnixMilli(dtt.taskState.StopTime),
		Result:      types.TaskResult(dtt.taskState.TaskResult), //nolint:gosec // ignore
		Logger:      logger.NewLogDBReader(dtt.database, dtt.taskState.RunID, dtt.taskState.TaskID),
	}

	if dtt.taskState.TaskError != "" {
		status.Error = fmt.Errorf("%v", dtt.taskState.TaskError)
	}

	return status
}

func (dtt *dbTestTask) GetTaskStatusVars() types.Variables {
	var statusVarsMap map[string]interface{}

	statusVars := vars.NewVariables(nil)

	err := yaml.Unmarshal([]byte(dtt.taskState.TaskStatus), &statusVarsMap)
	if err != nil {
		return statusVars
	}

	for key, value := range statusVarsMap {
		statusVars.SetVar(key, value)
	}

	return statusVars
}

func (dtt *dbTestTask) GetScopeOwner() types.TaskIndex {
	return types.TaskIndex(dtt.taskState.ScopeOwner)
}

func (dtt *dbTestTask) GetTaskResultUpdateChan(_ types.TaskResult) <-chan bool {
	return nil
}
