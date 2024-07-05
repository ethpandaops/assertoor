package types

import (
	"context"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
)

type TaskDescriptor struct {
	Name        string
	Description string
	Config      interface{}
	NewTask     func(ctx *TaskContext, options *TaskOptions) (Task, error)
}

type TaskOptions struct {
	// The name of the task to run.
	Name string `yaml:"name" json:"name"`
	// The configuration object of the task.
	Config *helper.RawMessage `yaml:"config" json:"config"`
	// The configuration settings to consume from runtime variables.
	ConfigVars map[string]string `yaml:"configVars" json:"configVars"`
	// The title of the task - this is used to describe the task to the user.
	Title string `yaml:"title" json:"title"`
	// Timeout defines the max time waiting for the condition to be met.
	Timeout helper.Duration `yaml:"timeout" json:"timeout"`
}

type TaskIndex uint64
type TaskResult uint8

const (
	TaskResultNone    TaskResult = 0
	TaskResultSuccess TaskResult = 1
	TaskResultFailure TaskResult = 2
)

type Task interface {
	Config() interface{}
	Timeout() time.Duration

	LoadConfig() error
	Execute(ctx context.Context) error
}

type TaskState interface {
	Index() TaskIndex
	ParentIndex() TaskIndex
	Name() string
	Title() string
	Description() string
	Config() interface{}
	Timeout() time.Duration
	GetTaskStatus() *TaskStatus
	GetTaskResultUpdateChan(oldResult TaskResult) <-chan bool
}

type TaskStatus struct {
	Index       TaskIndex
	ParentIndex TaskIndex
	IsStarted   bool
	IsRunning   bool
	StartTime   time.Time
	StopTime    time.Time
	Result      TaskResult
	Error       error
	Logger      *logger.LogScope
}

type TaskContext struct {
	Scheduler TaskScheduler
	Index     TaskIndex
	Vars      Variables
	Logger    *logger.LogScope
	NewTask   func(options *TaskOptions, variables Variables) (TaskIndex, error)
	SetResult func(result TaskResult)
}
