package test

import (
	"context"
	"fmt"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
	"github.com/sirupsen/logrus"
)

type Runnable interface {
	Validate() error
	Run(ctx context.Context) error
	Name() string
	Percent() float64
	Tasks() []task.Runnable
	ActiveTask() task.Runnable
}

type Test struct {
	name         string
	tasks        []task.Runnable
	cleanupTasks []task.Runnable
	log          logrus.FieldLogger
	config       Config

	activeTask task.Runnable
	currIndex  int
	metrics    Metrics
}

var _ Runnable = (*Test)(nil)

func AvailableTasks() task.MapOfRunnableInfo {
	return task.AvailableTasks()
}

func CreateRunnable(ctx context.Context, log logrus.FieldLogger, executionURL, consensusURL string, config Config) (Runnable, error) {
	runnable := &Test{
		name:      config.Name,
		tasks:     []task.Runnable{},
		log:       log.WithField("component", "test").WithField("test", config.Name),
		currIndex: 1,
		metrics:   NewMetrics("sync_test_coordinator", config.Name),
		config:    config,
	}

	runnable.metrics.Register()

	runnable.metrics.SetTestInfo(config.Name)
	runnable.metrics.SetTotalTasks(float64(len(config.Tasks)))

	for index, taskConfig := range config.Tasks {
		t, err := task.NewRunnableByName(ctx, log.WithField("component", "task"), executionURL, consensusURL, taskConfig.Name, taskConfig.Config, taskConfig.Title, taskConfig.Timeout.Duration)
		if err != nil {
			return nil, err
		}

		log.WithField("config", t.Config()).WithField("task", t.Name()).Info("created task")
		runnable.metrics.SetTaskInfo(t, index)

		runnable.tasks = append(runnable.tasks, t)
	}

	for index, taskConfig := range config.CleanupTasks {
		t, err := task.NewRunnableByName(ctx, log.WithField("component", "task"), executionURL, consensusURL, taskConfig.Name, taskConfig.Config, taskConfig.Title, taskConfig.Timeout.Duration)
		if err != nil {
			return nil, err
		}

		log.WithField("config", t.Config()).WithField("task", t.Name()).Info("created task")
		runnable.metrics.SetTaskInfo(t, len(config.Tasks)+index)

		runnable.cleanupTasks = append(runnable.cleanupTasks, t)
	}

	return runnable, nil
}

func (t *Test) Name() string {
	return t.name
}

func (t *Test) Validate() error {
	for _, task := range t.tasks {
		if err := task.ValidateConfig(); err != nil {
			return fmt.Errorf("task %s config validation failed: %s", task.Name(), err)
		}
	}

	for _, task := range t.cleanupTasks {
		if err := task.ValidateConfig(); err != nil {
			return fmt.Errorf("cleanup task %s config validation failed: %s", task.Name(), err)
		}
	}

	if len(t.tasks) == 0 {
		return fmt.Errorf("test %s has no tasks", t.name)
	}

	return nil
}

func (t *Test) Run(ctx context.Context) error {
	now := time.Now()

	defer t.metrics.SetTestDuration(float64(time.Since(now).Milliseconds()))
	defer t.runCleanupTasks(ctx)

	timeout := time.Hour * 24 * 365 // 1 year

	if t.config.Timeout.Duration > 0 {
		timeout = t.config.Timeout.Duration
	}

	t.log.WithField("timeout", timeout.String()).Info("setting test timeout")

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := t.runTasks(ctx)
	if err != nil {
		t.log.Info("test failed!")

		return err
	}

	t.log.Info("test completed!")

	return nil
}

func (t *Test) runCleanupTasks(ctx context.Context) {
	t.log.Info("running cleanup tasks..")

	for index, task := range t.cleanupTasks {
		if err := t.startTaskLoop(ctx, task, len(t.tasks)+index); err != nil {
			t.log.WithField("task", task.Name()).WithError(err).Error("failed to run cleanup task")
		}
	}
}

func (t *Test) runTasks(ctx context.Context) error {
	for index, task := range t.tasks {
		t.metrics.SetCurrentTask(task, index)

		if err := t.startTaskLoop(ctx, task, index); err != nil {
			return err
		}
	}

	return nil
}

func (t *Test) Percent() float64 {
	return float64(t.currIndex) / float64(len(t.tasks))
}

func (t *Test) Tasks() []task.Runnable {
	return t.tasks
}

func (t *Test) ActiveTask() task.Runnable {
	return t.activeTask
}

func (t *Test) startTaskLoop(ctx context.Context, ta task.Runnable, index int) error {
	t.log.WithField("task", ta.Name()).WithField("title", ta.Title()).Info("starting task")

	now := time.Now()

	defer func() {
		t.metrics.SetTaskDuration(ta, fmt.Sprintf("%d", index), float64(time.Since(now).Milliseconds()))
	}()

	t.activeTask = ta

	if err := t.runTask(ctx, ta); err != nil {
		return err
	}

	t.currIndex++

	t.log.WithField("task", ta.Name()).WithField("title", ta.Title()).Info("task completed!")

	return nil
}

func (t *Test) runTask(ctx context.Context, ta task.Runnable) error {
	if ta.Timeout() > 0 {
		cancellable, cancel := context.WithTimeout(ctx, ta.Timeout())
		defer cancel()

		t.log.WithField("name", ta.Name()).WithField("timeout", ta.Timeout()).Info("running task with timeout")

		ctx = cancellable
	}

	if err := ta.Start(ctx); err != nil {
		return err
	}

	if complete := t.tickTask(ctx, ta); complete {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(ta.PollingInterval()):
			if complete := t.tickTask(ctx, ta); complete {
				return nil
			}
		}
	}
}

func (t *Test) tickTask(ctx context.Context, ta task.Runnable) bool {
	log := t.log.WithField("name", ta.Name())

	log.Info(fmt.Sprintf("checking task for completion: (%s)", ta.Title()))

	complete, err := ta.IsComplete(ctx)

	log.WithFields(logrus.Fields{
		"complete": complete,
		"err":      err,
	}).Info("task status check")

	if err != nil {
		return false
	}

	if !complete {
		return false
	}

	t.log.Info("task is complete")

	return true
}
