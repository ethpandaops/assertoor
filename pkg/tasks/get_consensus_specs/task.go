package getconsensusspecs

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "get_consensus_specs"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Get consensus chain specs.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "specs",
				Type:        "object",
				Description: "The consensus chain specs object.",
			},
		},
		NewTask: NewTask,
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
	var specs map[string]interface{}

	for {
		specs = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().GetSpecValues()

		if specs != nil {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}

	t.ctx.Outputs.SetVar("specs", specs)

	t.ctx.ReportProgress(100, "Consensus specs retrieved")

	return nil
}
