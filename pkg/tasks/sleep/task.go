package sleep

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "sleep"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Sleeps for a specified duration.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	duration := t.config.Duration.Duration
	if duration <= 0 {
		return nil
	}

	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()

	t.ctx.ReportProgress(0, fmt.Sprintf("Sleeping for %v", duration))

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime)
			progress := float64(elapsed) / float64(duration) * 100

			if progress > 100 {
				progress = 100
			}

			remaining := duration - elapsed
			if remaining < 0 {
				remaining = 0
			}

			t.ctx.ReportProgress(progress, fmt.Sprintf("Sleeping... %v remaining", remaining.Round(time.Second)))
		case <-time.After(time.Until(startTime.Add(duration))):
			t.ctx.ReportProgress(100, "Sleep completed")

			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
