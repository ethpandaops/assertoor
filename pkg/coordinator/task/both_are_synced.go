package task

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type BothAreSynced struct {
	bundle    Bundle
	execution *ExecutionIsSynced
	consensus *ConsensusIsSynced
}

var _ Runnable = (*BothAreSynced)(nil)

const (
	NameBothAreSynced = "both_are_synced"
)

func NewBothAreSynced(ctx context.Context, bundle Bundle) *BothAreSynced {
	bundle.log = bundle.log.WithField("task", NameBothAreSynced)

	consensus := NewConsensusIsSynced(ctx, bundle)
	execution := NewExecutionIsSynced(ctx, bundle)

	return &BothAreSynced{
		bundle:    bundle,
		consensus: consensus,
		execution: execution,
	}
}

func (b *BothAreSynced) Name() string {
	return NameBothAreSynced
}

func (b *BothAreSynced) PollingInterval() time.Duration {
	return time.Second * 5
}

func (b *BothAreSynced) Start(ctx context.Context) error {
	return nil
}

func (b *BothAreSynced) Logger() logrus.FieldLogger {
	return b.bundle.Logger()
}

func (b *BothAreSynced) IsComplete(ctx context.Context) (bool, error) {
	execution, _ := b.execution.IsComplete(ctx)

	consensus, _ := b.consensus.IsComplete(ctx)

	if !consensus || !execution {
		return false, nil
	}

	return true, nil
}
