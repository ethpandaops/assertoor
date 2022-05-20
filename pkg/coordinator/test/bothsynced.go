package test

import (
	"context"
	"time"

	"github.com/samcm/sync-test-coordinator/pkg/coordinator/clients/consensus"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/task"
)

type BothSynced struct {
	bundle Bundle
	Tasks  []task.Runnable
}

var _ Runnable = (*BothSynced)(nil)

const (
	NameBothSynced = "both_synced"
)

func NewBothSynced(ctx context.Context, bundle Bundle) Runnable {
	return &BothSynced{
		bundle: bundle,
		Tasks: []task.Runnable{
			// Initialize
			task.NewSleep(ctx, *bundle.AsTaskBundle(), time.Second*1),

			// Check the clients for liveliness.
			task.NewExecutionIsHealthy(ctx, *bundle.AsTaskBundle()),
			task.NewConsensusIsHealthy(ctx, *bundle.AsTaskBundle()),

			// Wait until synced.
			task.NewBothAreSynced(ctx, *bundle.AsTaskBundle()),

			// Check the chains are progressing.
			task.NewConsensusCheckpointIsProgressing(ctx, *bundle.AsTaskBundle(), consensus.Head),
			task.NewConsensusCheckpointIsProgressing(ctx, *bundle.AsTaskBundle(), consensus.Finalized),
			task.NewExecutionIsProgressing(ctx, *bundle.AsTaskBundle()),
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
