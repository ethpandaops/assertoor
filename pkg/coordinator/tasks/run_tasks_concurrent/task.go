package runtasksconcurrent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/vars"
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
	tasks            []types.TaskIndex
	taskIdxMap       map[types.TaskIndex]int
	resultNotifyChan chan taskResultUpdate
}

type taskResultUpdate struct {
	task   types.TaskIndex
	result types.TaskResult
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:              ctx,
		options:          options,
		logger:           ctx.Logger.GetLogger(),
		taskIdxMap:       map[types.TaskIndex]int{},
		resultNotifyChan: make(chan taskResultUpdate, 100),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	// init child tasks
	childTasks := make([]types.TaskIndex, 0, len(config.Tasks))
	childScopes := vars.NewVariables(nil)

	for i := range config.Tasks {
		taskOpts, err := t.ctx.Scheduler.ParseTaskOptions(&config.Tasks[i])
		if err != nil {
			return fmt.Errorf("failed parsing child task config #%v : %w", i, err)
		}

		taskVars := t.ctx.Vars.NewScope()
		taskVars.SetVar("scopeOwner", uint64(t.ctx.Index))

		task, err := t.ctx.NewTask(taskOpts, taskVars)
		if err != nil {
			return fmt.Errorf("failed initializing child task #%v : %w", i, err)
		}

		childTasks = append(childTasks, task)

		childScopes.SetSubScope(fmt.Sprintf("%v", i), vars.NewScopeFilter(taskVars))
	}

	t.ctx.Outputs.SetSubScope("childScopes", childScopes)

	t.config = config
	t.tasks = childTasks

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

	completeChan := make(chan bool)
	go func() {
		taskWaitGroup.Wait()
		time.Sleep(100 * time.Millisecond)
		close(completeChan)
	}()

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

	resultMap := map[types.TaskIndex]types.TaskResult{}

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

			if !taskComplete {
				t.logger.Debugf("result update (%v success, %v failure)", successCount, failureCount)
			}
		case <-completeChan:
			if !taskComplete {
				taskComplete = true

				t.ctx.SetResult(types.TaskResultSuccess)
				t.logger.Infof("all child tasks completed (%v success, %v failure)", successCount, failureCount)
			}
		}
	}

	// cancel child context and wait for child tasks
	cancelFn()
	taskWaitGroup.Wait()

	return nil
}

func (t *Task) watchChildTask(_ context.Context, _ context.CancelFunc, task types.TaskIndex) {
	taskState := t.ctx.Scheduler.GetTaskState(task)
	oldStatus := types.TaskResultNone
	taskActive := true

	for taskActive {
		updateChan := taskState.GetTaskResultUpdateChan(oldStatus)
		if updateChan != nil {
			select {
			case <-t.taskCtx.Done():
				taskActive = false
			case <-time.After(10 * time.Second):
			case <-updateChan:
			}
		}

		taskStatus := taskState.GetTaskStatus()
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
