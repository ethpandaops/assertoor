package scheduler

import (
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"github.com/stretchr/testify/assert"
)

func newTestTaskState() *taskState {
	log := logger.NewLogger(&logger.ScopeOptions{
		Parent:      nil,
		HistorySize: 1000,
	})
	options := &types.TaskOptions{
		Name:   "sleep",
		Title:  "sleep task",
		Config: loadTestTaskConfig(`duration: 1ms`),
	}

	return &taskState{
		index:          1,
		options:        options,
		descriptor:     &types.TaskDescriptor{Name: "testTask"},
		taskVars:       vars.NewVariables(nil),
		logger:         log,
		taskOutputs:    vars.NewVariables(nil),
		taskStatusVars: vars.NewVariables(nil),
	}
}

func TestNewTaskState(t *testing.T) {
	ts := newTestTaskScheduler()
	options := &types.TaskOptions{Name: "sleep"}
	taskState, err := ts.newTaskState(options, nil, nil, false)

	assert.NoError(t, err)
	assert.NotNil(t, taskState)
	assert.Equal(t, options, taskState.options)
	assert.Equal(t, types.TaskIndex(1), taskState.index)
	assert.NotNil(t, taskState.logger)
	assert.NotNil(t, taskState.taskOutputs)
	assert.NotNil(t, taskState.taskStatusVars)
}

func TestSetTaskResult(t *testing.T) {
	ts := newTestTaskState()
	ts.setTaskResult(types.TaskResultSuccess, true)

	assert.Equal(t, types.TaskResultSuccess, ts.taskResult)
	assert.True(t, ts.updatedResult)
}

func TestGetTaskStatus(t *testing.T) {
	ts := newTestTaskState()
	ts.isStarted = true
	ts.isRunning = true
	ts.isSkipped = false
	ts.startTime = time.Now()
	ts.stopTime = time.Now().Add(1 * time.Second)
	ts.taskResult = types.TaskResultSuccess

	status := ts.GetTaskStatus()

	assert.Equal(t, ts.index, status.Index)
	assert.Equal(t, ts.isStarted, status.IsStarted)
	assert.Equal(t, ts.isRunning, status.IsRunning)
	assert.Equal(t, ts.isSkipped, status.IsSkipped)
	assert.Equal(t, ts.startTime, status.StartTime)
	assert.Equal(t, ts.stopTime, status.StopTime)
	assert.Equal(t, ts.taskResult, status.Result)
	assert.Nil(t, status.Error)
	assert.Equal(t, ts.logger, status.Logger)
}

func TestGetTaskStatusVars(t *testing.T) {
	ts := newTestTaskState()
	statusVars := ts.GetTaskStatusVars()

	assert.Equal(t, ts.taskStatusVars, statusVars)
}

func TestGetTaskVars(t *testing.T) {
	ts := newTestTaskState()
	taskVars := ts.GetTaskVars()

	assert.Equal(t, ts.taskVars, taskVars)
}

func TestGetTaskResultUpdateChan(t *testing.T) {
	ts := newTestTaskState()
	updateChan := ts.GetTaskResultUpdateChan(types.TaskResultNone)

	assert.NotNil(t, updateChan)

	ts.setTaskResult(types.TaskResultSuccess, true)

	recv := false
	select {
	case <-updateChan:
		recv = true
	default:
	}
	assert.True(t, recv)
}

func TestIndex(t *testing.T) {
	ts := newTestTaskState()
	index := ts.Index()

	assert.Equal(t, ts.index, index)
}

func TestParentIndex(t *testing.T) {
	ts := newTestTaskState()
	parentIndex := ts.ParentIndex()

	assert.Equal(t, types.TaskIndex(0), parentIndex)
}

func TestID(t *testing.T) {
	ts := newTestTaskState()
	id := ts.ID()

	assert.Equal(t, ts.options.ID, id)
}

func TestName(t *testing.T) {
	ts := newTestTaskState()
	name := ts.Name()

	assert.Equal(t, ts.options.Name, name)
}

func TestTitle(t *testing.T) {
	ts := newTestTaskState()
	title := ts.Title()

	assert.Equal(t, "sleep task", title)
}

func TestDescription(t *testing.T) {
	ts := newTestTaskState()
	description := ts.Description()

	assert.Equal(t, ts.descriptor.Description, description)
}

func TestConfig(t *testing.T) {
	ts := newTestTaskState()
	config := ts.Config()

	assert.Nil(t, config)
}

func TestTimeout(t *testing.T) {
	ts := newTestTaskState()
	timeout := ts.Timeout()

	assert.Equal(t, ts.options.Timeout.Duration, timeout)
}
