package task

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type SleepConfig struct {
	Duration time.Duration `yaml:"duration"`
}

type Sleep struct {
	bundle *Bundle
	config SleepConfig
	log    logrus.FieldLogger
}

var _ Runnable = (*Sleep)(nil)

const (
	NameSleep = "sleep"
)

func NewSleep(ctx context.Context, bundle *Bundle, config SleepConfig) *Sleep {
	return &Sleep{
		log:    bundle.log.WithField("task", NameSleep),
		bundle: bundle,
		config: config,
	}
}

func DefaultSleepConfig() SleepConfig {
	return SleepConfig{
		Duration: time.Second * 5,
	}
}

func (s *Sleep) Name() string {
	return NameSleep
}

func (s *Sleep) Config() interface{} {
	return s.config
}

func (s *Sleep) PollingInterval() time.Duration {
	return time.Second * 5
}

func (s *Sleep) Start(ctx context.Context) error {
	return nil
}

func (s *Sleep) Logger() logrus.FieldLogger {
	return s.log
}

func (s *Sleep) IsComplete(ctx context.Context) (bool, error) {
	s.log.Info("sleeping for ", s.config.Duration)

	time.Sleep(s.config.Duration)

	return true, nil
}
