package test

import (
	"context"

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
			task.NewConsensusIsHealthy(ctx, *bundle.AsTaskBundle()),
			// task.NewConsensusIsSynced(ctx, *bundle.AsTaskBundle()),
			task.NewConsensusCheckpointIsProgressing(ctx, *bundle.AsTaskBundle(), consensus.Head),
			task.NewConsensusCheckpointIsProgressing(ctx, *bundle.AsTaskBundle(), consensus.Finalized),
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
		b.bundle.Log.WithField("task", t.Name()).Info("running task")

		if err := RunTaskUntilCompletionOrError(ctx, b.bundle.Log, t); err != nil {
			return err
		}

		b.bundle.Log.WithField("task", t.Name()).Info("task complete")
	}

	return nil
}
