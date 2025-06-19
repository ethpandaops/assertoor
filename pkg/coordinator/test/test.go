package test

import (
	"context"
	"fmt"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/scheduler"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Test struct {
	runID         uint64
	services      types.TaskServices
	taskScheduler *scheduler.TaskScheduler
	logger        logrus.FieldLogger
	descriptor    types.TestDescriptor
	config        *types.TestConfig
	variables     types.Variables

	dbTestRun *db.TestRun

	status    types.TestStatus
	startTime time.Time
	stopTime  time.Time
	timeout   time.Duration
}

func CreateTest(runID uint64, descriptor types.TestDescriptor, logger logrus.FieldLogger, services types.TaskServices, configOverrides map[string]any) (types.TestRunner, error) {
	test := &Test{
		runID:      runID,
		services:   services,
		logger:     logger.WithField("RunID", runID).WithField("TestID", descriptor.ID()),
		descriptor: descriptor,
		config:     descriptor.Config(),
		status:     types.TestStatusPending,
	}
	if test.config.Timeout.Duration > 0 {
		test.timeout = test.config.Timeout.Duration
	}

	// set test variables
	test.variables = vars.NewVariables(descriptor.Vars())
	for cfgKey, cfgValue := range configOverrides {
		test.variables.SetVar(cfgKey, cfgValue)
	}

	// add test run to database
	configYaml, err := yaml.Marshal(test.variables.GetVarsMap(nil, false))
	if err != nil {
		return nil, err
	}

	test.dbTestRun = &db.TestRun{
		RunID:   runID,
		TestID:  descriptor.ID(),
		Name:    test.config.Name,
		Source:  descriptor.Source(),
		Config:  string(configYaml),
		Timeout: int32(test.timeout.Seconds()),
		Status:  string(test.status),
	}

	if err := services.Database().RunTransaction(func(tx *sqlx.Tx) error {
		err := services.Database().InsertTestRun(tx, test.dbTestRun)
		if err != nil {
			return err
		}

		return services.Database().SetAssertoorState(tx, "test.lastRunId", runID)
	}); err != nil {
		return nil, err
	}

	// parse tasks
	test.taskScheduler = scheduler.NewTaskScheduler(test.logger, services, test.variables, runID)
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

func (t *Test) updateTestStatus() error {
	// update test run in database
	t.dbTestRun.Status = string(t.status)

	if t.startTime.IsZero() {
		t.dbTestRun.StartTime = 0
	} else {
		t.dbTestRun.StartTime = t.startTime.UnixMilli()
	}

	if t.stopTime.IsZero() {
		t.dbTestRun.StopTime = 0
	} else {
		t.dbTestRun.StopTime = t.stopTime.UnixMilli()
	}

	if err := t.services.Database().RunTransaction(func(tx *sqlx.Tx) error {
		return t.services.Database().UpdateTestRunStatus(tx, t.dbTestRun)
	}); err != nil {
		return err
	}

	return nil
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

	if err := t.updateTestStatus(); err != nil {
		t.logger.WithError(err).Error("failed updating test status")
	}

	defer func() {
		t.stopTime = time.Now()

		if err := t.updateTestStatus(); err != nil {
			t.logger.WithError(err).Error("failed updating test status")
		}
	}()

	// run test tasks
	t.logger.WithField("timeout", t.timeout.String()).Info("starting test")

	err := t.taskScheduler.RunTasks(ctx, t.timeout)

	if ctx.Err() != nil {
		t.logger.Info("test aborted!")
		t.status = types.TestStatusAborted

		return fmt.Errorf("test aborted")
	}

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
	t.logger.Info("aborting test")
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
