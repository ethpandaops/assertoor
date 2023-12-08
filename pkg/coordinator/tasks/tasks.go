package tasks

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"

	checkclientsarehealthy "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensusblockproposals "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_block_proposals"
	checkconsensussyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_sync_status"
	checkexecutionsyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_execution_sync_status"
	runcommand "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_command"
	runshell "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_shell"
	runtasks "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_tasks"
	runtasksconcurrent "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_tasks_concurrent"
	sleep "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/sleep"
)

var AvailableTaskDescriptors = []*types.TaskDescriptor{
	checkclientsarehealthy.TaskDescriptor,
	checkconsensusblockproposals.TaskDescriptor,
	checkconsensussyncstatus.TaskDescriptor,
	checkexecutionsyncstatus.TaskDescriptor,
	runcommand.TaskDescriptor,
	runshell.TaskDescriptor,
	runtasks.TaskDescriptor,
	runtasksconcurrent.TaskDescriptor,
	sleep.TaskDescriptor,
}
