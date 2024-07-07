package types

import (
	"context"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/sirupsen/logrus"
)

type TestStatus string

const (
	TestStatusPending TestStatus = "pending"
	TestStatusRunning TestStatus = "running"
	TestStatusSuccess TestStatus = "success"
	TestStatusFailure TestStatus = "failure"
	TestStatusSkipped TestStatus = "skipped"
	TestStatusAborted TestStatus = "aborted"
)

type Test interface {
	Validate() error
	Run(ctx context.Context) error
	RunID() uint64
	TestID() string
	Name() string
	StartTime() time.Time
	StopTime() time.Time
	Timeout() time.Duration
	Percent() float64
	Status() TestStatus
	Logger() logrus.FieldLogger
	AbortTest(skipCleanup bool)
	GetTaskScheduler() TaskScheduler
	GetTestVariables() Variables
}

type TestConfig struct {
	ID           string                 `yaml:"id" json:"id"`
	Name         string                 `yaml:"name" json:"name"`
	Timeout      helper.Duration        `yaml:"timeout" json:"timeout"`
	Config       map[string]interface{} `yaml:"config" json:"config"`
	ConfigVars   map[string]string      `yaml:"configVars" json:"configVars"`
	Tasks        []helper.RawMessage    `yaml:"tasks" json:"tasks"`
	CleanupTasks []helper.RawMessage    `yaml:"cleanupTasks" json:"cleanupTasks"`
	Schedule     *TestSchedule          `yaml:"schedule" json:"schedule"`
}

type ExternalTestConfig struct {
	ID         string                 `yaml:"id" json:"id"`
	File       string                 `yaml:"file" json:"file"`
	Name       string                 `yaml:"name" json:"name"`
	Timeout    *helper.Duration       `yaml:"timeout" json:"timeout"`
	Config     map[string]interface{} `yaml:"config" json:"config"`
	ConfigVars map[string]string      `yaml:"configVars" json:"configVars"`
	Schedule   *TestSchedule          `yaml:"schedule" json:"schedule"`
}

type TestSchedule struct {
	Startup bool     `yaml:"startup" json:"startup"`
	Cron    []string `yaml:"cron" json:"cron"`
}

type TestDescriptor interface {
	ID() string
	Source() string
	Config() *TestConfig
	Vars() Variables
	Err() error
}
