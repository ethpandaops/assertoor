package scheduler

import (
	"fmt"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
)

func (ts *TaskScheduler) ParseTaskOptions(rawtask helper.IRawMessage) (*types.TaskOptions, error) {
	options := &types.TaskOptions{}
	if err := rawtask.Unmarshal(&options); err != nil {
		return nil, fmt.Errorf("error parsing task: %w", err)
	}

	return options, nil
}
