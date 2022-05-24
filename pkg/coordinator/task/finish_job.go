package task

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type FinishJob struct {
	log    logrus.FieldLogger
	config FinishJobConfig
	bundle *Bundle
}

var _ Runnable = (*FinishJob)(nil)

const (
	NameFinishJob = "finish_job"
)

func NewFinishJob(ctx context.Context, bundle *Bundle) *FinishJob {
	return &FinishJob{
		log:    bundle.log.WithField("task", NameFinishJob),
		config: bundle.TaskConfig.FinishJob,
	}
}

func (c *FinishJob) Name() string {
	return NameFinishJob
}

func (c *FinishJob) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *FinishJob) Start(ctx context.Context) error {
	if len(c.config.Command) != 0 {
		cmd := NewRunCommand(ctx, c.bundle, c.config.Command...)

		if err := cmd.Start(ctx); err != nil {
			return err
		}

		return nil
	}

	return nil
}

func (c *FinishJob) Logger() logrus.FieldLogger {
	return c.log
}

func (c *FinishJob) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
