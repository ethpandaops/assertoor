package test

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
	"github.com/ethpandaops/minccino/pkg/coordinator/task/scheduler"
	"github.com/sirupsen/logrus"
)

type Runnable interface {
	Validate() error
	Run(ctx context.Context) error
	Name() string
	Percent() float64
}

type Test struct {
	name          string
	taskScheduler *scheduler.TaskScheduler

	log    logrus.FieldLogger
	config Config

	metrics Metrics
}

func CreateRunnable(ctx context.Context, log logrus.FieldLogger, clientPool *clients.ClientPool, config Config) (Runnable, error) {
	runnable := &Test{
		name:    config.Name,
		log:     log.WithField("component", "test").WithField("test", config.Name),
		config:  config,
		metrics: NewMetrics("sync_test_coordinator", config.Name),
	}

	// parse tasks
	runnable.taskScheduler = scheduler.NewTaskScheduler(runnable.log, clientPool)
	for _, rawtask := range config.Tasks {
		taskOptions, err := runnable.taskScheduler.ParseTaskOptions(&rawtask)
		if err != nil {
			return nil, err
		}
		_, err = runnable.taskScheduler.AddRootTask(taskOptions)
		if err != nil {
			return nil, err
		}
	}

	for _, rawtask := range config.CleanupTasks {
		taskOptions, err := runnable.taskScheduler.ParseTaskOptions(&rawtask)
		if err != nil {
			return nil, err
		}
		_, err = runnable.taskScheduler.AddCleanupTask(taskOptions)
		if err != nil {
			return nil, err
		}
	}

	// setup metrics
	runnable.metrics.Register()

	runnable.metrics.SetTestInfo(config.Name)
	runnable.metrics.SetTotalTasks(float64(len(config.Tasks)))

	return runnable, nil
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
