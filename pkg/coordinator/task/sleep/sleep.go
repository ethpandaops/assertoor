package sleep

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

type Task struct {
	config  Config
	log     logrus.FieldLogger
	title   string
	timeout time.Duration
}

const (
	Name        = "sleep"
	Description = "Sleeps for a specified duration."
)

func NewTask(ctx context.Context, log logrus.FieldLogger, config Config, title string, timeout time.Duration) *Task {
	return &Task{
		log:     log.WithField("task", Name),
		config:  config,
		title:   title,
		timeout: timeout,
	}
}

func (t *Task) Name() string {
	return Name
}

func (t *Task) Title() string {
	return t.title
}

func (t *Task) Description() string {
	return Description
}

func (t *Task) Timeout() time.Duration {
	return t.timeout
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

	return nil
}

func (t *Task) PollingInterval() time.Duration {
	return time.Second * 5
}

func (t *Task) Start(ctx context.Context) error {
	return nil
}

func (t *Task) Cleanup(ctx context.Context) error {
	return nil
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.log
}

func (t *Task) IsComplete(ctx context.Context) (bool, error) {
	t.log.Info("sleeping for ", t.config.Duration)

	time.Sleep(t.config.Duration.Duration)

	return true, nil
}
