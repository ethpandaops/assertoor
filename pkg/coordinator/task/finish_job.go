package task //nolint:dupl // false positive

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type FinishJob struct {
	bundle *Bundle
}

var _ Runnable = (*FinishJob)(nil)

const (
	NameFinishJob = "finish_job"
)

func NewFinishJob(ctx context.Context, bundle *Bundle) *FinishJob {
	bundle.log = bundle.log.WithField("task", NameFinishJob)

	return &FinishJob{
		bundle: bundle,
	}
}

func (c *FinishJob) Name() string {
	return NameFinishJob
}

func (c *FinishJob) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *FinishJob) Start(ctx context.Context) error {
	if len(c.bundle.TaskConfig.FinishJob.Command) != 0 {
		cmd := NewRunCommand(ctx, c.bundle, c.bundle.TaskConfig.FinishJob.Command...)

		if err := cmd.Start(ctx); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (c *FinishJob) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *FinishJob) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
