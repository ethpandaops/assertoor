package test

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

type Test struct {
	name          string
	taskScheduler *TaskScheduler

	log    logrus.FieldLogger
	config Config

	metrics Metrics
}

func CreateRunnable(ctx context.Context, coordinator types.Coordinator, config Config) (types.Test, error) {
	test := &Test{
		name:    config.Name,
		log:     coordinator.Logger().WithField("component", "test").WithField("test", config.Name),
		config:  config,
		metrics: NewMetrics("sync_test_coordinator", config.Name),
	}

	// parse tasks
	test.taskScheduler = NewTaskScheduler(test.log, coordinator)
	for _, rawtask := range config.Tasks {
		taskOptions, err := test.taskScheduler.ParseTaskOptions(&rawtask)
		if err != nil {
			return nil, err
		}
		_, err = test.taskScheduler.AddRootTask(taskOptions)
		if err != nil {
			return nil, err
		}
	}

	for _, rawtask := range config.CleanupTasks {
		taskOptions, err := test.taskScheduler.ParseTaskOptions(&rawtask)
		if err != nil {
			return nil, err
		}
		_, err = test.taskScheduler.AddCleanupTask(taskOptions)
		if err != nil {
			return nil, err
		}
	}

	// setup metrics
	test.metrics.Register()

	test.metrics.SetTestInfo(config.Name)
	test.metrics.SetTotalTasks(float64(len(config.Tasks)))

	return test, nil
}

func (t *Test) Name() string {
	return t.name
}

func (t *Test) Validate() error {
	err := t.taskScheduler.ValidateTaskConfigs()
	if err != nil {
		return fmt.Errorf("test %s config validation failed: %w", t.name, err)
	}

	if t.taskScheduler.GetTaskCount() == 0 {
		return fmt.Errorf("test %s has no tasks", t.name)
	}

	return nil
}

func (t *Test) Run(ctx context.Context) error {
	now := time.Now()

	defer t.metrics.SetTestDuration(float64(time.Since(now).Milliseconds()))

	timeout := time.Hour * 24 * 365 // 1 year
	if t.config.Timeout.Duration > 0 {
		timeout = t.config.Timeout.Duration
	}
	t.log.WithField("timeout", timeout.String()).Info("setting test timeout")

	err := t.taskScheduler.RunTasks(ctx, timeout)
	if err != nil {
		t.log.Info("test failed!")

		return err
	}

	t.log.Info("test completed!")

	return nil
}

func (t *Test) Percent() float64 {
	//return float64(t.currIndex) / float64(len(t.tasks))
	return 0
}
