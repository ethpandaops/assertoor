package tasks

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"

	checkclientsarehealthy "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensusattestationstats "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_attestation_stats"
	checkconsensusblockproposals "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_block_proposals"
	checkconsensusfinality "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_finality"
	checkconsensusforks "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_forks"
	checkconsensusproposerduty "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_proposer_duty"
	checkconsensusreorgs "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_reorgs"
	checkconsensusslotrange "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_slot_range"
	checkconsensussyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_sync_status"
	checkconsensusvalidatorstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_validator_status"
	checkexecutionsyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_execution_sync_status"
	generateblobtransactions "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_blob_transactions"
	generateblschanges "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_bls_changes"
	generatechildwallet "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_child_wallet"
	generatedeposits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_deposits"
	generateeoatransactions "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_eoa_transactions"
	generateexits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_exits"
	generateslashings "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_slashings"
	generatetransaction "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_transaction"
	runcommand "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_command"
	runshell "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_shell"
	runtaskbackground "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_task_background"
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
	checkconsensusforks.TaskDescriptor,
	checkconsensusproposerduty.TaskDescriptor,
	checkconsensusreorgs.TaskDescriptor,
	checkconsensusslotrange.TaskDescriptor,
	checkconsensussyncstatus.TaskDescriptor,
	checkconsensusvalidatorstatus.TaskDescriptor,
	checkexecutionsyncstatus.TaskDescriptor,
	generateblobtransactions.TaskDescriptor,
	generateblschanges.TaskDescriptor,
	generatechildwallet.TaskDescriptor,
	generateeoatransactions.TaskDescriptor,
	generatedeposits.TaskDescriptor,
	generateexits.TaskDescriptor,
	generateslashings.TaskDescriptor,
	generatetransaction.TaskDescriptor,
	runcommand.TaskDescriptor,
	runshell.TaskDescriptor,
	runtaskbackground.TaskDescriptor,
	runtaskmatrix.TaskDescriptor,
	runtaskoptions.TaskDescriptor,
	runtasks.TaskDescriptor,
	runtasksconcurrent.TaskDescriptor,
	sleep.TaskDescriptor,
}
