package tasks

import (
	"github.com/ethpandaops/minccino/pkg/coordinator/types"

	awaitconsensusblockproposal "github.com/ethpandaops/minccino/pkg/coordinator/tasks/await_consensus_block_proposal"
	checkclientsarehealthy "github.com/ethpandaops/minccino/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensussyncstatus "github.com/ethpandaops/minccino/pkg/coordinator/tasks/check_consensus_sync_status"
	checkexecutionsyncstatus "github.com/ethpandaops/minccino/pkg/coordinator/tasks/check_execution_sync_status"
	runcommand "github.com/ethpandaops/minccino/pkg/coordinator/tasks/run_command"
	runtasks "github.com/ethpandaops/minccino/pkg/coordinator/tasks/run_tasks"
	runtasksconcurrent "github.com/ethpandaops/minccino/pkg/coordinator/tasks/run_tasks_concurrent"
	sleep "github.com/ethpandaops/minccino/pkg/coordinator/tasks/sleep"
)

var AvailableTaskDescriptors = []*types.TaskDescriptor{
	awaitconsensusblockproposal.TaskDescriptor,
	checkclientsarehealthy.TaskDescriptor,
	checkconsensussyncstatus.TaskDescriptor,
	checkexecutionsyncstatus.TaskDescriptor,
	runcommand.TaskDescriptor,
	runtasks.TaskDescriptor,
	runtasksconcurrent.TaskDescriptor,
	sleep.TaskDescriptor,
}
