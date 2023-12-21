package tasks

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"

	checkclientsarehealthy "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensusattestationstats "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_attestation_stats"
	checkconsensusblockproposals "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_block_proposals"
	checkconsensusfinality "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_finality"
	checkconsensusproposerduty "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_proposer_duty"
	checkconsensusreorgs "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_reorgs"
	checkconsensussyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_sync_status"
	checkconsensusvalidatorstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_validator_status"
	checkexecutionsyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_execution_sync_status"
	generateblschanges "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_bls_changes"
	generatedeposits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_deposits"
	generateexits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_exits"
	generateslashings "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_slashings"
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
	checkconsensusattestationstats.TaskDescriptor,
	checkconsensusblockproposals.TaskDescriptor,
	checkconsensusfinality.TaskDescriptor,
	checkconsensusproposerduty.TaskDescriptor,
	checkconsensusreorgs.TaskDescriptor,
	checkconsensussyncstatus.TaskDescriptor,
	checkconsensusvalidatorstatus.TaskDescriptor,
	checkexecutionsyncstatus.TaskDescriptor,
	generateblschanges.TaskDescriptor,
	generatedeposits.TaskDescriptor,
	generateexits.TaskDescriptor,
	generateslashings.TaskDescriptor,
	runcommand.TaskDescriptor,
	runshell.TaskDescriptor,
	runtaskmatrix.TaskDescriptor,
	runtaskoptions.TaskDescriptor,
	runtasks.TaskDescriptor,
	runtasksconcurrent.TaskDescriptor,
	sleep.TaskDescriptor,
}
