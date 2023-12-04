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
	log           logrus.FieldLogger
	config        *Config
	metrics       Metrics

	status    types.TestStatus
	startTime time.Time
	stopTime  time.Time
	timeout   time.Duration
}

func CreateTest(ctx context.Context, coordinator types.Coordinator, config *Config) (types.Test, error) {
	test := &Test{
		name:    config.Name,
		log:     coordinator.Logger().WithField("component", "test").WithField("test", config.Name),
		config:  config,
		metrics: NewMetrics("sync_test_coordinator", config.Name),
	}
	if test.config.Timeout.Duration > 0 {
		test.timeout = test.config.Timeout.Duration
	}

	if config.Disable {
		test.status = types.TestStatusSkipped
	} else {

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
	}
	return test, nil
}

func (t *Test) Name() string {
	return t.name
}

func (t *Test) StartTime() time.Time {
	return t.startTime
}

func (t *Test) StopTime() time.Time {
	return t.stopTime
}

func (t *Test) Timeout() time.Duration {
	return t.timeout
}

func (t *Test) Status() types.TestStatus {
	return t.status
}

func (t *Test) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Test) Validate() error {
	if t.taskScheduler == nil {
		return nil
	}
	err := t.taskScheduler.ValidateTaskConfigs()
	if err != nil {
		t.status = types.TestStatusFailure
		return fmt.Errorf("test %s config validation failed: %w", t.name, err)
	}

	if t.taskScheduler.GetTaskCount() == 0 {
		t.status = types.TestStatusFailure
		return fmt.Errorf("test %s has no tasks", t.name)
	}

	return nil
}

func (t *Test) Run(ctx context.Context) error {
	if t.taskScheduler == nil {
		return nil
	}
	if t.status != types.TestStatusPending {
		return fmt.Errorf("test has already been started")
	}

	t.startTime = time.Now()
	t.status = types.TestStatusRunning

	defer func() {
		t.metrics.SetTestDuration(float64(time.Since(t.startTime).Milliseconds()))
		t.stopTime = time.Now()
	}()

	t.log.WithField("timeout", t.timeout.String()).Info("starting test")

	err := t.taskScheduler.RunTasks(ctx, t.timeout)
	if err != nil {
		t.log.Info("test failed!")
		t.status = types.TestStatusFailure
		return err
	}

	t.log.Info("test completed!")
	t.status = types.TestStatusSuccess
	return nil
}

func (t *Test) GetTaskScheduler() types.TaskScheduler {
	return t.taskScheduler
}

func (t *Test) Percent() float64 {
	//return float64(t.currIndex) / float64(len(t.tasks))
	return 0
}
