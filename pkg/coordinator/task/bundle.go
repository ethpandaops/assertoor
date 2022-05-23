package task

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/execution"
	"github.com/sirupsen/logrus"
)

// Bundle holds all possible configuration for a task.
type Bundle struct {
	log          logrus.FieldLogger
	ConsensusURL string
	ExecutionURL string
	TaskConfig   Config
}

func NewBundle(log logrus.FieldLogger, consensusURL, executionURL string, config Config) *Bundle {
	return &Bundle{
		log:          log,
		ConsensusURL: consensusURL,
		ExecutionURL: executionURL,
		TaskConfig:   config,
	}
}

func (b *Bundle) Logger() logrus.FieldLogger {
	return b.log
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

func (b *Bundle) GetExecutionClient(ctx context.Context) *execution.Client {
	client := execution.NewExecutionClient(b.log, b.ExecutionURL)
	if err := client.Bootstrap(ctx); err != nil {
		b.log.WithError(err).Error("failed to bootstrap execution client")
	}

	for !client.Bootstrapped() {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second * 3):
			if err := client.Bootstrap(ctx); err != nil {
				b.log.WithError(err).Error("failed to bootstrap execution client")
			}
		}
	}

	return &client
}
