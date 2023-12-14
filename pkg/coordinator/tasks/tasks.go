package tasks

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"

	checkclientsarehealthy "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensusblockproposals "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_block_proposals"
	checkconsensusfinality "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_finality"
	checkconsensussyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_sync_status"
	checkexecutionsyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_execution_sync_status"
	generateblschanges "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_bls_changes"
	generatedeposits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_deposits"
	generateexits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_exits"
	runcommand "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_command"
	runshell "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_shell"
	runtaskmatrix "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_task_matrix"
	runtaskoptions "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_task_options"
	runtasks "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_tasks"
	runtasksconcurrent "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_tasks_concurrent"
	sleep "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/sleep"
)

var AvailableTaskDescriptors = []*types.TaskDescriptor{
	checkclientsarehealthy.TaskDescriptor,
	checkconsensusblockproposals.TaskDescriptor,
	checkconsensusfinality.TaskDescriptor,
	checkconsensussyncstatus.TaskDescriptor,
	checkexecutionsyncstatus.TaskDescriptor,
	generateblschanges.TaskDescriptor,
	generatedeposits.TaskDescriptor,
	generateexits.TaskDescriptor,
	runcommand.TaskDescriptor,
	runshell.TaskDescriptor,
	runtaskmatrix.TaskDescriptor,
	runtaskoptions.TaskDescriptor,
	runtasks.TaskDescriptor,
	runtasksconcurrent.TaskDescriptor,
	sleep.TaskDescriptor,
}
