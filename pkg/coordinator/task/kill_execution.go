package task //nolint:dupl // false positive

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type KillExecution struct {
	bundle *Bundle
}

var _ Runnable = (*KillExecution)(nil)

const (
	NameKillExecution = "kill_execution"
)

func NewKillExecution(ctx context.Context, bundle *Bundle) *KillExecution {
	bundle.log = bundle.log.WithField("task", NameKillExecution)

	return &KillExecution{
		bundle: bundle,
	}
}

func (c *KillExecution) Name() string {
	return NameKillExecution
}

func (c *KillExecution) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *KillExecution) Start(ctx context.Context) error {
	if len(c.bundle.TaskConfig.KillExecution.Command) != 0 {
		cmd := NewRunCommand(ctx, c.bundle, c.bundle.TaskConfig.KillExecution.Command...)

		if err := cmd.Start(ctx); err != nil {
			return err
		}

		return nil
	}

	for _, client := range ExecutionClientNames {
		cmd := NewRunCommand(ctx, c.bundle, "pkill", client)

		if err := cmd.Start(ctx); err != nil {
			c.bundle.log.WithError(err).Error("Failed to run kill execution command")
		}
	}

	return nil
}

func (c *KillExecution) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *KillExecution) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
