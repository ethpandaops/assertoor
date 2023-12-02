package tasks

import (
	"github.com/ethpandaops/minccino/pkg/coordinator/task/types"

	runcommand "github.com/ethpandaops/minccino/pkg/coordinator/task/tasks/run_command"
	sleep "github.com/ethpandaops/minccino/pkg/coordinator/task/tasks/sleep"
)

var AvailableTaskDescriptors = []*types.TaskDescriptor{
	runcommand.TaskDescriptor,
	sleep.TaskDescriptor,
}
