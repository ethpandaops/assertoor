package test

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/db"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/tasks"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"gopkg.in/yaml.v3"
)

type dbTest struct {
	database *db.Database

	runID   int
	testRun *db.TestRun

	taskIndexMtx sync.Mutex
	taskIndex    []*db.TaskStateIndex
}

func LoadTestFromDB(database *db.Database, runID int) (types.Test, error) {
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
	return uint64(dbt.runID)
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

func (dbt *dbTest) GetTaskCount() int {
	dbt.loadTaskIndex()
	return len(dbt.taskIndex)
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

func (dbt *dbTest) GetTaskState(taskIndex types.TaskIndex) types.TaskState {
	task, err := dbt.database.GetTaskStateByTaskID(dbt.runID, int(taskIndex))
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
		Result:      types.TaskResult(dtt.taskState.TaskResult),
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
