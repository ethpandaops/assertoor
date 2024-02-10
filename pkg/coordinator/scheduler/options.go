package scheduler

import (
	"fmt"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
	"github.com/ethpandaops/assertoor/pkg/coordinator/human-duration"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"gopkg.in/yaml.v3"
)

func (ts *TaskScheduler) ParseTaskOptions(rawtask *helper.RawMessage) (*types.TaskOptions, error) {
	options := &types.TaskOptions{}
	if err := rawtask.Unmarshal(&options); err != nil {
		return nil, fmt.Errorf("error parsing task: %w", err)
	}

	return options, nil
}

type TaskOptsSettings struct {
	// The title of the task - this is used to describe the task to the user.
	Title string `yaml:"title" json:"title"`
	// The configuration settings to consume from runtime variables.
	ConfigVars map[string]string `yaml:"configVars" json:"configVars"`
	// Timeout defines the max time waiting for the condition to be met.
	Timeout human.Duration `yaml:"timeout" json:"timeout"`
}

func (ts *TaskScheduler) NewTaskOptions(task *types.TaskDescriptor, config interface{}, settings *TaskOptsSettings) (*types.TaskOptions, error) {
	options := &types.TaskOptions{
		Name: task.Name,
	}

	if settings != nil {
		options.Title = settings.Title
		options.ConfigVars = settings.ConfigVars
		options.Timeout = settings.Timeout
	}

	if config != nil {
		configYaml, err := yaml.Marshal(config)
		if err != nil {
			return nil, fmt.Errorf("error serializing task config: %w", err)
		}

		err = yaml.Unmarshal(configYaml, options.Config)
		if err != nil {
			return nil, fmt.Errorf("error parsing task config: %w", err)
		}
	}

	return options, nil
}
