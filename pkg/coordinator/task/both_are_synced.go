package task

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type BothAreSyncedConfig struct {
	ConsensusIsSyncedConfig ConsensusIsSyncedConfig `yaml:"consensus"`
	ExecutionIsSyncedConfig ExecutionIsSyncedConfig `yaml:"execution"`
}

type BothAreSynced struct {
	log       logrus.FieldLogger
	execution *ExecutionIsSynced
	consensus *ConsensusIsSynced
	config    BothAreSyncedConfig
}

var _ Runnable = (*BothAreSynced)(nil)

const (
	NameBothAreSynced = "both_are_synced"
)

func NewBothAreSynced(ctx context.Context, bundle *Bundle, config BothAreSyncedConfig) *BothAreSynced {
	consensus := NewConsensusIsSynced(ctx, bundle, config.ConsensusIsSyncedConfig)
	execution := NewExecutionIsSynced(ctx, bundle, config.ExecutionIsSyncedConfig)

	return &BothAreSynced{
		log:       bundle.log.WithField("task", NameBothAreSynced),
		consensus: consensus,
		execution: execution,
		config:    config,
	}
}

func DefaultBothAreSyncedConfig() BothAreSyncedConfig {
	return BothAreSyncedConfig{
		ConsensusIsSyncedConfig: DefaultConsensusIsSyncedConfig(),
		ExecutionIsSyncedConfig: DefaultExecutionIsSyncedConfig(),
	}
}

func (b *BothAreSynced) Config() interface{} {
	return b.config
}

func (b *BothAreSynced) Name() string {
	return NameBothAreSynced
}

func (b *BothAreSynced) PollingInterval() time.Duration {
	return time.Second * 5
}

func (b *BothAreSynced) Start(ctx context.Context) error {
	if err := b.consensus.Start(ctx); err != nil {
		return err
	}

	if err := b.execution.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (b *BothAreSynced) Logger() logrus.FieldLogger {
	return b.log
}

func (b *BothAreSynced) IsComplete(ctx context.Context) (bool, error) {
	execution, _ := b.execution.IsComplete(ctx)

	consensus, _ := b.consensus.IsComplete(ctx)

	if !consensus || !execution {
		return false, nil
	}

	return true, nil
}
