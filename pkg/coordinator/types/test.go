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

type TestRunner interface {
	Test
	Validate() error
	Run(ctx context.Context) error
	Logger() logrus.FieldLogger
	GetTestVariables() Variables
}

type Test interface {
	RunID() uint64
	TestID() string
	Name() string
	StartTime() time.Time
	StopTime() time.Time
	Timeout() time.Duration
	Status() TestStatus
	GetTaskScheduler() TaskScheduler
	AbortTest(skipCleanup bool)
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
	Startup   bool     `yaml:"startup" json:"startup"`
	Cron      []string `yaml:"cron" json:"cron"`
	SkipQueue bool     `yaml:"skipQueue" json:"skipQueue"`
}

type TestDescriptor interface {
	ID() string
	Source() string
	BasePath() string
	Config() *TestConfig
	Vars() Variables
	Err() error
}
