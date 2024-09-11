package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
	"gopkg.in/yaml.v2"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewTaskScheduler(t *testing.T) {
	logger := logrus.New()
	services := NewServicesProvider(nil, nil, nil)
	variables := vars.NewVariables(nil)

	ts := NewTaskScheduler(logger, services, variables)

	assert.NotNil(t, ts)
	assert.Equal(t, logger, ts.logger)
	assert.Equal(t, services, ts.services)
	assert.Equal(t, variables, ts.rootVars)
	assert.Equal(t, types.TaskIndex(1), ts.nextTaskIndex)
	assert.Empty(t, ts.rootTasks)
	assert.Empty(t, ts.allTasks)
	assert.Empty(t, ts.taskStateMap)
}

func newTestTaskScheduler() *TaskScheduler {
	logger := logrus.New()
	logger.Level = 0
	services := NewServicesProvider(nil, nil, nil)
	variables := vars.NewVariables(nil)

	return NewTaskScheduler(logger, services, variables)
}

func loadTestTaskConfig(config string) *helper.RawMessage {
	configRaw := helper.RawMessage{}

	err := yaml.Unmarshal([]byte(config), &configRaw)
	if err != nil {
		panic(err)
	}

	return &configRaw
}

func TestAddRootTask(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name: "sleep",
	}
	taskIndex, err := ts.AddRootTask(options)

	assert.NoError(t, err)
	assert.Equal(t, types.TaskIndex(1), taskIndex)
	assert.Contains(t, ts.rootTasks, taskIndex)
}

func TestAddCleanupTask(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name: "sleep",
	}
	taskIndex, err := ts.AddCleanupTask(options)

	assert.NoError(t, err)
	assert.Equal(t, types.TaskIndex(1), taskIndex)
	assert.Contains(t, ts.rootCleanupTasks, taskIndex)
}

func TestRunTasks(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 10ms`),
	}
	_, _ = ts.AddRootTask(options)
	_, _ = ts.AddCleanupTask(options)

	ctx := context.Background()
	t1 := time.Now()

	err := ts.RunTasks(ctx, 0)
	assert.NoError(t, err)

	delay := time.Since(t1)
	assert.True(t, delay >= 20*time.Millisecond)
	assert.True(t, delay < 25*time.Millisecond)
}

func TestCancelTasks(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 1s`),
	}
	_, err := ts.AddRootTask(options)
	assert.NoError(t, err)

	ctx := context.Background()
	t1 := time.Now()

	go func() {
		time.Sleep(20 * time.Millisecond)
		ts.CancelTasks(true)
	}()

	err = ts.RunTasks(ctx, 0)
	assert.Error(t, err)

	delay := time.Since(t1)
	assert.True(t, delay < 25*time.Millisecond)
}

func TestGetServices(t *testing.T) {
	ts := newTestTaskScheduler()

	assert.Equal(t, ts.services, ts.GetServices())
}

func TestParseTaskOptions(t *testing.T) {
	ts := newTestTaskScheduler()

	rawTask := loadTestTaskConfig(`
name: sleep
title: "test sleep"
timeout: 2s
config:
  duration: 1s
configVars:
  test1: "test2"
`)
	options, err := ts.ParseTaskOptions(rawTask)

	assert.NoError(t, err)
	assert.NotNil(t, options)
	assert.Equal(t, "sleep", options.Name)
	assert.Equal(t, "test sleep", options.Title)
	assert.Equal(t, 2*time.Second, options.Timeout.Duration)
	assert.NotNil(t, options.Config)
	assert.NotNil(t, options.ConfigVars)
	assert.Equal(t, "test2", options.ConfigVars["test1"])
}

func TestGetAllTasks(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 1s`),
	}
	_, _ = ts.AddRootTask(options)

	allTasks := ts.GetAllTasks()
	assert.NotEmpty(t, allTasks)
}

func TestGetTaskCount(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 1s`),
	}
	_, _ = ts.AddRootTask(options)

	taskCount := ts.GetTaskCount()
	assert.Equal(t, 1, taskCount)
}

func TestGetRootTasks(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 1s`),
	}
	_, _ = ts.AddRootTask(options)

	rootTasks := ts.GetRootTasks()
	assert.NotEmpty(t, rootTasks)
}

func TestGetAllCleanupTasks(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 1s`),
	}
	_, _ = ts.AddCleanupTask(options)

	allCleanupTasks := ts.GetAllCleanupTasks()
	assert.NotEmpty(t, allCleanupTasks)
}

func TestGetRootCleanupTasks(t *testing.T) {
	ts := newTestTaskScheduler()

	options := &types.TaskOptions{
		Name:   "sleep",
		Config: loadTestTaskConfig(`duration: 1s`),
	}
	_, _ = ts.AddCleanupTask(options)

	rootCleanupTasks := ts.GetRootCleanupTasks()
	assert.NotEmpty(t, rootCleanupTasks)
}
