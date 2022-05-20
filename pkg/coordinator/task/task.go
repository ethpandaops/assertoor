package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
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

// GetConsensusClient returns a new consensus client. Useful for not having to worry about bootstrapping.
func (b *Bundle) GetConsensusClient(ctx context.Context) *consensus.Client {
	client := consensus.NewConsensusClient(b.log, b.ConsensusURL)
	if err := client.Bootstrap(ctx); err != nil {
		b.log.WithError(err).Error("failed to bootstrap consensus client")
	}

	for !client.Bootstrapped() {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second * 3):
			if err := client.Bootstrap(ctx); err != nil {
				b.log.WithError(err).Error("failed to bootstrap consensus client")
			}
		}
	}

	return &client
}
