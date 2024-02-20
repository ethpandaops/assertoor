package scheduler

import (
	"fmt"

	"github.com/ethpandaops/assertoor/pkg/coordinator/helper"
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

func GetRawConfig(config interface{}) *helper.RawMessage {
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		return nil
	}

	configRaw := helper.RawMessage{}

	err = yaml.Unmarshal(configYaml, &configRaw)
	if err != nil {
		return nil
	}

	return &configRaw
}
