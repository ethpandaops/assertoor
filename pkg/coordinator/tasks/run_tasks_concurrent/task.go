package runtasksconcurrent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "run_tasks_concurrent"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Runs multiple tasks in parallel.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx              *types.TaskContext
	options          *types.TaskOptions
	config           Config
	logger           logrus.FieldLogger
	taskCtx          context.Context
	tasks            []types.Task
	taskIdxMap       map[types.Task]int
	resultNotifyChan chan taskResultUpdate
}

type taskResultUpdate struct {
	task   types.Task
	result types.TaskResult
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	config := DefaultConfig()

	if options.Config != nil {
		conf := &Config{}
		if err := options.Config.Unmarshal(&conf); err != nil {
			return nil, fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}

		if err := mergo.Merge(&config, conf, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging task config for %v: %w", TaskName, err)
		}
	}

	childTasks := []types.Task{}

	for i := range config.Tasks {
		taskOpts, err := ctx.Scheduler.ParseTaskOptions(&config.Tasks[i])
		if err != nil {
			return nil, fmt.Errorf("failed parsing child task config #%v : %w", i, err)
		}

		task, err := ctx.NewTask(taskOpts)
		if err != nil {
			return nil, fmt.Errorf("failed initializing child task #%v : %w", i, err)
		}

		childTasks = append(childTasks, task)
	}

	return &Task{
		ctx:              ctx,
		options:          options,
		config:           config,
		logger:           ctx.Scheduler.GetLogger().WithField("task", TaskName),
		tasks:            childTasks,
		taskIdxMap:       map[types.Task]int{},
		resultNotifyChan: make(chan taskResultUpdate, 100),
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.options.Title
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	taskWaitGroup := sync.WaitGroup{}

	taskCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	t.taskCtx = taskCtx

	// start child tasks
	for i := range t.tasks {
		taskWaitGroup.Add(1)

		t.taskIdxMap[t.tasks[i]] = i

		go func(i int) {
			defer taskWaitGroup.Done()

			task := t.tasks[i]

			t.logger.Debugf("starting child task %v", i)

			//nolint:errcheck // ignore
			t.ctx.Scheduler.ExecuteTask(taskCtx, task, t.watchChildTask)
		}(i)
	}

	// watch result updates
	successLimit := t.config.SucceedTaskCount
	if successLimit == 0 {
		successLimit = uint64(len(t.tasks))
	}

	failureLimit := t.config.FailTaskCount
	if failureLimit == 0 {
		failureLimit = uint64(len(t.tasks))
	}

	var successCount, failureCount, pendingCount uint64

	resultMap := map[types.Task]types.TaskResult{}

	taskComplete := false
	for !taskComplete {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case resultUpdate := <-t.resultNotifyChan:
			resultMap[resultUpdate.task] = resultUpdate.result

			successCount = 0
			failureCount = 0
			pendingCount = 0

			for _, task := range t.tasks {
				result := resultMap[task]
				switch result {
				case types.TaskResultSuccess:
					successCount++
				case types.TaskResultFailure:
					failureCount++
				case types.TaskResultNone:
					pendingCount++
				}
			}

			if successCount >= successLimit {
				t.logger.Infof("success limit reached (%v success, %v failure)", successCount, failureCount)
				t.ctx.SetResult(types.TaskResultSuccess)

				taskComplete = true
			}

			if !taskComplete && failureCount >= failureLimit {
				t.logger.Infof("failure limit reached (%v success, %v failure)", successCount, failureCount)
				t.ctx.SetResult(types.TaskResultFailure)

				taskComplete = true
			}

			if !taskComplete && pendingCount == 0 {
				t.logger.Infof("all child tasks completed (%v success, %v failure)", successCount, failureCount)
				t.ctx.SetResult(types.TaskResultFailure)

				taskComplete = true
			}

			if !taskComplete {
				t.logger.Debugf("result update (%v success, %v failure)", successCount, failureCount)
			}
		}
	}

	// cancel child context and wait for child tasks
	cancelFn()
	taskWaitGroup.Wait()

	return nil
}

func (t *Task) watchChildTask(_ context.Context, _ context.CancelFunc, task types.Task) {
	oldStatus := types.TaskResultNone
	taskActive := true

	for taskActive {
		updateChan := t.ctx.Scheduler.GetTaskResultUpdateChan(task, oldStatus)
		if updateChan != nil {
			select {
			case <-t.taskCtx.Done():
				taskActive = false
			case <-time.After(10 * time.Second):
			case <-updateChan:
			}
		}

		taskStatus := t.ctx.Scheduler.GetTaskStatus(task)
		if !taskStatus.IsRunning {
			taskActive = false
		}

		if taskStatus.Result == oldStatus {
			continue
		}

		t.logger.Debugf("result update notification for task %v (%v -> %v)", t.taskIdxMap[task], oldStatus, taskStatus.Result)

		t.resultNotifyChan <- taskResultUpdate{
			task:   task,
			result: taskStatus.Result,
		}

		oldStatus = taskStatus.Result
	}
}
