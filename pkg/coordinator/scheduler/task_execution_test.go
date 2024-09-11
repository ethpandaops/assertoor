package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/stretchr/testify/assert"
)

func TestExecuteTask(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 10ms`),
	}
	taskIndex, err := ts.AddRootTask(options)
	assert.NoError(t, err)

	ctx := context.Background()
	err = ts.ExecuteTask(ctx, taskIndex, nil)
	assert.NoError(t, err)

	taskState := ts.getTaskState(taskIndex)
	assert.NotNil(t, taskState)
	assert.True(t, taskState.isStarted)
	assert.False(t, taskState.isRunning)
	assert.Equal(t, types.TaskResultSuccess, taskState.taskResult)
}

func TestExecuteTaskWithCondition(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name: "sleep",
		If:   "|false",
	}
	taskIndex, err := ts.AddRootTask(options)
	assert.NoError(t, err)

	ctx := context.Background()
	err = ts.ExecuteTask(ctx, taskIndex, nil)
	assert.NoError(t, err)

	taskState := ts.getTaskState(taskIndex)
	assert.NotNil(t, taskState)
	assert.True(t, taskState.isSkipped)
	assert.Equal(t, types.TaskResultNone, taskState.taskResult)
}

func TestExecuteTaskWithTimeout(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:    "sleep",
		Config:  loadTestTaskConfig(`duration: 1s`),
		Timeout: helper.Duration{Duration: 10 * time.Millisecond},
	}
	taskIndex, err := ts.AddRootTask(options)
	assert.NoError(t, err)

	ctx := context.Background()
	err = ts.ExecuteTask(ctx, taskIndex, nil)
	assert.Error(t, err)

	taskState := ts.getTaskState(taskIndex)
	assert.NotNil(t, taskState)
	assert.True(t, taskState.isTimeout)
	assert.Equal(t, types.TaskResultFailure, taskState.taskResult)
}

func TestExecuteTaskWithPanic(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name: "sleep",
	}
	taskIndex, err := ts.AddRootTask(options)
	assert.NoError(t, err)

	// Mock task to panic
	taskState := ts.getTaskState(taskIndex)
	oldLoader := taskState.descriptor.NewTask
	taskState.descriptor.NewTask = func(_ *types.TaskContext, _ *types.TaskOptions) (types.Task, error) {
		return &mockTask{panicOnExecute: true}, nil
	}

	defer func() {
		taskState.descriptor.NewTask = oldLoader
	}()

	ctx := context.Background()
	err = ts.ExecuteTask(ctx, taskIndex, nil)
	assert.NoError(t, err)

	taskState = ts.getTaskState(taskIndex)
	assert.NotNil(t, taskState)
	assert.Equal(t, types.TaskResultFailure, taskState.taskResult)
}

func TestWatchTaskPass(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name: "sleep",
	}
	taskIndex, err := ts.AddRootTask(options)
	assert.NoError(t, err)

	// Mock task to success
	taskState := ts.getTaskState(taskIndex)
	oldLoader := taskState.descriptor.NewTask
	taskState.descriptor.NewTask = func(_ *types.TaskContext, _ *types.TaskOptions) (types.Task, error) {
		return &mockTask{}, nil
	}

	defer func() {
		taskState.descriptor.NewTask = oldLoader
	}()

	ctx, cancel := context.WithCancel(context.Background())
	go ts.WatchTaskPass(ctx, cancel, taskIndex)

	taskState = ts.getTaskState(taskIndex)
	taskState.setTaskResult(types.TaskResultSuccess, false)

	select {
	case <-ctx.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("context was not cancelled")
	}
}

type mockTask struct {
	panicOnExecute bool
}

func (m *mockTask) LoadConfig() error {
	return nil
}

func (m *mockTask) Execute(_ context.Context) error {
	if m.panicOnExecute {
		panic(errors.New("mock panic"))
	}

	return nil
}

func (m *mockTask) Config() interface{} {
	return nil
}

func (m *mockTask) Timeout() time.Duration {
	return 0
}
