package task

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Runnable represents an INDIVIDUAL task to be run. These tasks should be as small as possible.
type Runnable interface {
	Start(ctx context.Context) error
	IsComplete(ctx context.Context) (bool, error)

	Name() string
	PollingInterval() time.Duration
	Logger() logrus.FieldLogger
}
