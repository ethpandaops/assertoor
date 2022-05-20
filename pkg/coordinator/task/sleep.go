package task

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type Sleep struct {
	bundle   Bundle
	duration time.Duration
}

var _ Runnable = (*Sleep)(nil)

const (
	NameSleep = "sleep"
)

func NewSleep(ctx context.Context, bundle Bundle, duration time.Duration) *Sleep {
	bundle.log = bundle.log.WithField("task", NameSleep)

	return &Sleep{
		bundle:   bundle,
		duration: duration,
	}
}

func (s *Sleep) Name() string {
	return NameSleep
}

func (s *Sleep) PollingInterval() time.Duration {
	return time.Second * 5
}

func (s *Sleep) Start(ctx context.Context) error {
	return nil
}

func (s *Sleep) Logger() logrus.FieldLogger {
	return s.bundle.Logger()
}

func (s *Sleep) IsComplete(ctx context.Context) (bool, error) {
	s.Logger().Info("sleeping for", s.duration)

	time.Sleep(s.duration)

	return true, nil
}
