package scheduler

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
)

// ExecuteTask executes a task
// this function blocks until the task is executed or the context cancelled
func (ts *TaskScheduler) ExecuteTask(ctx context.Context, taskIndex types.TaskIndex, taskWatchFn func(ctx context.Context, cancelFn context.CancelFunc, taskIndex types.TaskIndex)) error {
	taskState := ts.getTaskState(taskIndex)
	if taskState == nil {
		return fmt.Errorf("task not found")
	}

	taskLogger := taskState.logger.GetLogger()

	// check if task has already been started/executed
	if taskState.isStarted {
		return fmt.Errorf("task has already been executed")
	}

	taskState.isStarted = true
	taskState.startTime = time.Now()
	taskState.isRunning = true
	taskState.taskStatusVars.SetVar("started", true)
	taskState.taskStatusVars.SetVar("running", true)

	if err := taskState.updateTaskState(); err != nil {
		taskLogger.Errorf("task state update on db failed: %v", err)
	}

	defer func() {
		taskState.isRunning = false
		taskState.stopTime = time.Now()
		taskState.taskStatusVars.SetVar("running", false)

		if err := taskState.updateTaskState(); err != nil {
			taskLogger.Errorf("task state update on db failed: %v", err)
		}

		taskState.logger.Flush()
	}()

	// check task condition if defined
	if taskState.options.If != "" {
		conditionResult, _, err := taskState.taskVars.ResolveQuery(taskState.options.If)
		if err != nil {
			taskLogger.Errorf("task condition evaluation failed: %v", err)
			taskState.setTaskResult(types.TaskResultFailure, false)

			return fmt.Errorf("task condition evaluation failed: %w", err)
		}

		isValid, isOk := conditionResult.(bool)
		if !isOk {
			taskLogger.Warnf("task condition is not a boolean: %v", conditionResult)
		}

		if !isValid {
			taskLogger.Infof("task condition not met, skipping task")

			taskState.isSkipped = true
			taskState.setTaskResult(types.TaskResultNone, false)

			return nil
		}
	}

	// create task control context
	taskCtx := &types.TaskContext{
		Scheduler: ts,
		Index:     taskState.index,
		Vars:      taskState.taskVars,
		Outputs:   taskState.taskOutputs,
		Logger:    taskState.logger,
		NewTask: func(options *types.TaskOptions, variables types.Variables) (types.TaskIndex, error) {
			task, err := ts.newTaskState(options, taskState, variables, taskState.isCleanup)
			if err != nil {
				return 0, err
			}

			return task.index, nil
		},
		SetResult: func(result types.TaskResult) {
			taskState.setTaskResult(result, true)
		},
	}

	// create task instance
	task, err := taskState.descriptor.NewTask(taskCtx, taskState.options)
	if err != nil {
		return fmt.Errorf("failed task '%v' initialization: %w", taskState.options.Name, err)
	}

	// load task config
	err = task.LoadConfig()
	if err != nil {
		taskLogger.Errorf("config validation failed: %v", err)
		taskState.setTaskResult(types.TaskResultFailure, false)

		return fmt.Errorf("task %v config validation failed: %w", taskState.Name(), err)
	}

	taskState.task = task
	taskState.taskConfig = task.Config()

	defer func() {
		taskState.task = nil
	}()

	// create cancelable task context
	taskContext, taskCancelFn := context.WithCancel(ctx)
	taskTimeout := task.Timeout()

	if taskTimeout > 0 {
		go func() {
			select {
			case <-time.After(taskTimeout):
				taskState.isTimeout = true
				taskState.taskStatusVars.SetVar("timeout", true)

				taskLogger.Warnf("task timed out")
				taskCancelFn()
			case <-taskContext.Done():
			}
		}()
	}

	defer taskCancelFn()

	defer func() {
		if r := recover(); r != nil {
			pErr, ok := r.(error)
			if ok {
				taskState.taskError = pErr
			}

			taskLogger.Errorf("task execution panic: %v, stack: %v", r, string(debug.Stack()))
			taskState.setTaskResult(types.TaskResultFailure, false)
		}
	}()

	// run task watcher if supplied
	if taskWatchFn != nil {
		go taskWatchFn(taskContext, taskCancelFn, taskState.index)
	}

	// execute task
	taskLogger.Infof("starting task")

	err = task.Execute(taskContext)
	if err != nil {
		taskLogger.Errorf("task execution returned error: %v", err)

		if taskState.taskError == nil {
			taskState.taskError = err
		}
	}

	// set task result
	if !taskState.updatedResult || taskState.taskResult == types.TaskResultNone {
		// set task result if not already done by task
		if taskState.isTimeout || err != nil {
			taskState.setTaskResult(types.TaskResultFailure, false)
		} else {
			taskState.setTaskResult(types.TaskResultSuccess, false)
		}
	}

	if taskState.taskResult == types.TaskResultFailure {
		taskLogger.Warnf("task failed with failure result: %v", taskState.taskError)
		return fmt.Errorf("task failed: %w", taskState.taskError)
	}

	taskLogger.Infof("task completed")

	return nil
}

func (ts *TaskScheduler) WatchTaskPass(ctx context.Context, cancelFn context.CancelFunc, taskIndex types.TaskIndex) {
	taskState := ts.GetTaskState(taskIndex)

	// poll task result and cancel context when task result is passed or failed
	for {
		updateChan := taskState.GetTaskResultUpdateChan(types.TaskResultNone)
		if updateChan != nil {
			select {
			case <-ctx.Done():
				return
			case <-updateChan:
			}
		}

		taskStatus := taskState.GetTaskStatus()
		if taskStatus.Result != types.TaskResultNone {
			cancelFn()
			return
		}
	}
}
