package test

import (
	"context"
	"errors"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
	"github.com/sirupsen/logrus"
)

type Runnable interface {
	Init(ctx context.Context) error
	Run(ctx context.Context) error

	Name() string
}

var (
	ErrTestNotFound = errors.New("test not found")
)

func NewTestByName(ctx context.Context, name string, bundle Bundle) (Runnable, error) {
	bundle.Log = bundle.Log.WithField("test", name)

	switch name {
	case NameBothSynced:
		return NewBothSynced(ctx, bundle), nil
	default:
		return nil, ErrTestNotFound
	}
}

func RunUntilCompletionOrError(ctx context.Context, test Runnable) error {
	if err := test.Init(ctx); err != nil {
		return err
	}

	return test.Run(ctx)
}

func RunTaskUntilCompletionOrError(ctx context.Context, log logrus.FieldLogger, runnable task.Runnable) error {
	log = log.WithField("task", runnable.Name())

	if complete := tickTask(ctx, log, runnable); complete {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(runnable.PollingInterval()):
			if complete := tickTask(ctx, log, runnable); complete {
				return nil
			}
		}
	}
}

func tickTask(ctx context.Context, log logrus.FieldLogger, runnable task.Runnable) bool {
	log.Info("checking task for completion")

	complete, err := runnable.IsComplete(ctx)

	log.WithFields(logrus.Fields{
		"complete": complete,
		"err":      err,
	}).Info("task status check")

	if err != nil || !complete {
		return false
	}

	log.Info("task is complete")

	return true
}
