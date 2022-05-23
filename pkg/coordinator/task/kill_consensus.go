package task //nolint:dupl // false positive

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type KillConsensus struct {
	bundle *Bundle
}

var _ Runnable = (*KillConsensus)(nil)

const (
	NameKillConsensus = "kill_consensus"
)

func NewKillConsensus(ctx context.Context, bundle *Bundle) *KillConsensus {
	bundle.log = bundle.log.WithField("task", NameKillConsensus)

	return &KillConsensus{
		bundle: bundle,
	}
}

func (c *KillConsensus) Name() string {
	return NameKillConsensus
}

func (c *KillConsensus) PollingInterval() time.Duration {
	return time.Second * 1
}

func (c *KillConsensus) Start(ctx context.Context) error {
	if len(c.bundle.TaskConfig.KillConsensus.Command) != 0 {
		cmd := NewRunCommand(ctx, c.bundle, c.bundle.TaskConfig.KillConsensus.Command...)

		if err := cmd.Start(ctx); err != nil {
			return err
		}

		return nil
	}

	for _, client := range ConsensusClientNames {
		cmd := NewRunCommand(ctx, c.bundle, "pkill", client)

		if err := cmd.Start(ctx); err != nil {
			c.bundle.log.WithError(err).Error("Failed to run kill consensus command")
		}
	}

	return nil
}

func (c *KillConsensus) Logger() logrus.FieldLogger {
	return c.bundle.Logger()
}

func (c *KillConsensus) IsComplete(ctx context.Context) (bool, error) {
	return true, nil
}
