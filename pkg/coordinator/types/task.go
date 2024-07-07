package types

import (
	"context"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/logger"
	"github.com/sirupsen/logrus"
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

type TaskResult uint8

const (
	TaskResultNone    TaskResult = 0
	TaskResultSuccess TaskResult = 1
	TaskResultFailure TaskResult = 2
)

type Task interface {
	Name() string
	Title() string
	Description() string

	Config() interface{}
	Logger() logrus.FieldLogger
	Timeout() time.Duration

	LoadConfig() error
	Execute(ctx context.Context) error
}

type TaskStatus struct {
	Index       uint64
	ParentIndex uint64
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
	Index     uint64
	Vars      Variables
	Logger    *logger.LogScope
	NewTask   func(options *TaskOptions, variables Variables) (Task, error)
	SetResult func(result TaskResult)
}
