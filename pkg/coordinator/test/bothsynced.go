package test

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
)

type BothSynced struct {
	bundle *Bundle
	Tasks  []task.Runnable
}

var _ Runnable = (*BothSynced)(nil)

const (
	NameBothSynced = "both_synced"
)

func NewBothSynced(ctx context.Context, bundle *Bundle) Runnable {
	return &BothSynced{
		bundle: bundle,
		Tasks: []task.Runnable{
			// Initialize
			task.NewSleep(ctx, bundle.AsTaskBundle(), time.Second*15),

			// Check the clients for liveliness.
			task.NewExecutionIsHealthy(ctx, bundle.AsTaskBundle()),
			task.NewConsensusIsHealthy(ctx, bundle.AsTaskBundle()),

			// Wait until synced.
			task.NewBothAreSynced(ctx, bundle.AsTaskBundle()),

			// // Check the chains are progressing.
			task.NewConsensusCheckpointIsProgressing(ctx, bundle.AsTaskBundle(), consensus.Head),
			task.NewConsensusCheckpointIsProgressing(ctx, bundle.AsTaskBundle(), consensus.Finalized),
			task.NewExecutionIsProgressing(ctx, bundle.AsTaskBundle()),

			// Kill the clients.
			task.NewKillConsensus(ctx, bundle.AsTaskBundle()),
			task.NewKillExecution(ctx, bundle.AsTaskBundle()),

			// Ensure they're dead.
			task.NewConsensusIsUnhealthy(ctx, bundle.AsTaskBundle()),
			task.NewExecutionIsUnhealthy(ctx, bundle.AsTaskBundle()),

			// Run any cleanup.
			task.NewFinishJob(ctx, bundle.AsTaskBundle()),

			// Sleep for a little while to give time for metrics to be reported.
			task.NewSleep(ctx, bundle.AsTaskBundle(), time.Second*30),
		},
	}
}

func (b *BothSynced) Name() string {
	return NameBothSynced
}

func (b *BothSynced) Init(ctx context.Context) error {
	return nil
}

func (b *BothSynced) Run(ctx context.Context) error {
	for _, t := range b.Tasks {
		if err := RunTaskUntilCompletionOrError(ctx, b.bundle.Log, t); err != nil {
			return err
		}
	}

	return nil
}
