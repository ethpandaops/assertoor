package types

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type TestStatus uint8

const (
	TestStatusPending TestStatus = 0
	TestStatusRunning TestStatus = 1
	TestStatusSuccess TestStatus = 2
	TestStatusFailure TestStatus = 3
	TestStatusSkipped TestStatus = 4
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
