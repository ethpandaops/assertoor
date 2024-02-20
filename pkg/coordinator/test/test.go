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
	runID         uint64
	taskScheduler *scheduler.TaskScheduler
	logger        logrus.FieldLogger
	descriptor    types.TestDescriptor
	config        *types.TestConfig
	variables     types.Variables

	status    types.TestStatus
	startTime time.Time
	stopTime  time.Time
	timeout   time.Duration
}

func CreateTest(runID uint64, descriptor types.TestDescriptor, logger logrus.FieldLogger, services types.TaskServices, variables types.Variables) (types.Test, error) {
	test := &Test{
		runID:      runID,
		logger:     logger.WithField("RunID", runID),
		descriptor: descriptor,
		config:     descriptor.Config(),
		status:     types.TestStatusPending,
	}
	if test.config.Timeout.Duration > 0 {
		test.timeout = test.config.Timeout.Duration
	}

	// set test variables
	test.variables = variables.NewScope()
	for name, value := range test.config.Config {
		test.variables.SetVar(name, value)
	}

	err := test.variables.CopyVars(variables, test.config.ConfigVars)
	if err != nil {
		return nil, err
	}

	// parse tasks
	test.taskScheduler = scheduler.NewTaskScheduler(test.logger, services, test.variables)
	for i := range test.config.Tasks {
		taskOptions, err := test.taskScheduler.ParseTaskOptions(&test.config.Tasks[i])
		if err != nil {
			return nil, err
		}

		_, err = test.taskScheduler.AddRootTask(taskOptions)
		if err != nil {
			return nil, err
		}
	}

	for i := range test.config.CleanupTasks {
		taskOptions, err := test.taskScheduler.ParseTaskOptions(&test.config.CleanupTasks[i])
		if err != nil {
			return nil, err
		}

		_, err = test.taskScheduler.AddCleanupTask(taskOptions)
		if err != nil {
			return nil, err
		}
	}

	return test, nil
}

func (t *Test) RunID() uint64 {
	return t.runID
}

func (t *Test) TestID() string {
	return t.descriptor.ID()
}

func (t *Test) Name() string {
	return t.config.Name
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
		return fmt.Errorf("test %s has no tasks", t.config.Name)
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

	if t.status == types.TestStatusAborted {
		return nil
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

	if t.status == types.TestStatusAborted {
		t.logger.Info("test aborted!")

		return fmt.Errorf("test aborted")
	}

	if err != nil {
		t.logger.Info("test failed!")
		t.status = types.TestStatusFailure

		return err
	}

	t.logger.Info("test completed!")
	t.status = types.TestStatusSuccess

	return nil
}

func (t *Test) AbortTest(skipCleanup bool) {
	t.status = types.TestStatusAborted

	if t.taskScheduler != nil {
		t.taskScheduler.CancelTasks(skipCleanup)
	}
}

func (t *Test) GetTaskScheduler() types.TaskScheduler {
	return t.taskScheduler
}

func (t *Test) GetTestVariables() types.Variables {
	return t.variables
}

func (t *Test) Percent() float64 {
	return 0
}
