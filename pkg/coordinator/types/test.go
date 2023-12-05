package types

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type TestStatus string

const (
	TestStatusPending TestStatus = "pending"
	TestStatusRunning TestStatus = "running"
	TestStatusSuccess TestStatus = "success"
	TestStatusFailure TestStatus = "failure"
	TestStatusSkipped TestStatus = "skipped"
)

type Test interface {
	Validate() error
	Run(ctx context.Context) error
	Name() string
	StartTime() time.Time
	StopTime() time.Time
	Timeout() time.Duration
	Percent() float64
	Status() TestStatus
	Logger() logrus.FieldLogger
	GetTaskScheduler() TaskScheduler
}
