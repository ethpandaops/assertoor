package test

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/scheduler"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

type Test struct {
	name          string
	taskScheduler *scheduler.TaskScheduler
	logger        logrus.FieldLogger
	config        *Config

	status    types.TestStatus
	startTime time.Time
	stopTime  time.Time
	timeout   time.Duration
}

func CreateTest(coordinator types.Coordinator, config *Config, variables types.Variables) (types.Test, error) {
	test := &Test{
		name:   config.Name,
		logger: coordinator.Logger().WithField("component", "test").WithField("test", config.Name),
		config: config,
		status: types.TestStatusPending,
	}
	if test.config.Timeout.Duration > 0 {
		test.timeout = test.config.Timeout.Duration
	}

	if config.Disable {
		test.status = types.TestStatusSkipped
	} else {
		// set test variables
		testVars := variables.NewScope()
		for name, value := range config.Config {
			testVars.SetVar(name, value)
		}

		testVars.CopyVars(variables, config.ConfigVars)

		// parse tasks
		test.taskScheduler = scheduler.NewTaskScheduler(test.logger, coordinator, testVars)
		for i := range config.Tasks {
			taskOptions, err := test.taskScheduler.ParseTaskOptions(&config.Tasks[i])
			if err != nil {
				return nil, err
			}

			_, err = test.taskScheduler.AddRootTask(taskOptions)
			if err != nil {
				return nil, err
			}
		}

		for i := range config.CleanupTasks {
			taskOptions, err := test.taskScheduler.ParseTaskOptions(&config.CleanupTasks[i])
			if err != nil {
				return nil, err
			}

			_, err = test.taskScheduler.AddCleanupTask(taskOptions)
			if err != nil {
				return nil, err
			}
		}
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
	return t.logger
}

func (t *Test) Validate() error {
	if t.taskScheduler == nil {
		return nil
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

	// track start/stop time
	t.startTime = time.Now()
	t.status = types.TestStatusRunning

	defer func() {
		t.stopTime = time.Now()
	}()

	// run test tasks
	t.logger.WithField("timeout", t.timeout.String()).Info("starting test")

	err := t.taskScheduler.RunTasks(ctx, t.timeout)
	if err != nil {
		t.logger.Info("test failed!")
		t.status = types.TestStatusFailure

		return err
	}

	t.logger.Info("test completed!")
	t.status = types.TestStatusSuccess

	return nil
}

func (t *Test) GetTaskScheduler() types.TaskScheduler {
	return t.taskScheduler
}

func (t *Test) Percent() float64 {
	return 0
}
