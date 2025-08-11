package tasks

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"

	checkclientsarehealthy "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensusattestationstats "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_attestation_stats"
	checkconsensusblockproposals "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_block_proposals"
	checkconsensusfinality "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_finality"
	checkconsensusforks "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_forks"
	checkconsensusidentity "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_identity"
	checkconsensusproposerduty "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_proposer_duty"
	checkconsensusreorgs "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_reorgs"
	checkconsensusslotrange "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_slot_range"
	checkconsensussyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_sync_status"
	checkconsensusvalidatorstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_consensus_validator_status"
	checkethcall "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_eth_call"
	checkexecutionsyncstatus "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/check_execution_sync_status"
	generateblobtransactions "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_blob_transactions"
	generateblschanges "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_bls_changes"
	generatechildwallet "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_child_wallet"
	generateconsolidations "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_consolidations"
	generatedeposits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_deposits"
	generateeoatransactions "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_eoa_transactions"
	generateexits "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_exits"
	generateslashings "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_slashings"
	generatetransaction "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_transaction"
	generatewithdrawalrequests "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/generate_withdrawal_requests"
	getconsensusspecs "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/get_consensus_specs"
	getconsensusvalidators "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/get_consensus_validators"
	checkexecutionblock "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/get_execution_block"
	getpubkeysfrommnemonic "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/get_pubkeys_from_mnemonic"
	getrandommnemonic "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/get_random_mnemonic"
	getwalletdetails "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/get_wallet_details"
	runcommand "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_command"
	runexternaltasks "github.com/ethpandaops/assertoor/pkg/coordinator/tasks/run_external_tasks"
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
	checkconsensusidentity.TaskDescriptor,
	checkconsensusproposerduty.TaskDescriptor,
	checkconsensusreorgs.TaskDescriptor,
	checkconsensusslotrange.TaskDescriptor,
	checkconsensussyncstatus.TaskDescriptor,
	checkconsensusvalidatorstatus.TaskDescriptor,
	checkexecutionblock.TaskDescriptor,
	checkethcall.TaskDescriptor,
	checkexecutionsyncstatus.TaskDescriptor,
	generateblobtransactions.TaskDescriptor,
	generateblschanges.TaskDescriptor,
	generatechildwallet.TaskDescriptor,
	generateconsolidations.TaskDescriptor,
	generateeoatransactions.TaskDescriptor,
	generatedeposits.TaskDescriptor,
	generateexits.TaskDescriptor,
	generateslashings.TaskDescriptor,
	generatetransaction.TaskDescriptor,
	generatewithdrawalrequests.TaskDescriptor,
	getpubkeysfrommnemonic.TaskDescriptor,
	getconsensusspecs.TaskDescriptor,
	getconsensusvalidators.TaskDescriptor,
	getrandommnemonic.TaskDescriptor,
	getwalletdetails.TaskDescriptor,
	runcommand.TaskDescriptor,
	runexternaltasks.TaskDescriptor,
	runshell.TaskDescriptor,
	runtaskbackground.TaskDescriptor,
	runtaskmatrix.TaskDescriptor,
	runtaskoptions.TaskDescriptor,
	runtasks.TaskDescriptor,
	runtasksconcurrent.TaskDescriptor,
	sleep.TaskDescriptor,
}

func GetTaskDescriptor(name string) *types.TaskDescriptor {
	// lookup task descriptor by name
	var taskDescriptor *types.TaskDescriptor

	for _, taskDesc := range AvailableTaskDescriptors {
		if taskDesc.Name == name {
			taskDescriptor = taskDesc
			break
		}

		if len(taskDesc.Aliases) > 0 {
			isAlias := false

			for _, alias := range taskDesc.Aliases {
				if alias == name {
					isAlias = true
					break
				}
			}

			if isAlias {
				taskDescriptor = taskDesc
				break
			}
		}
	}

	return taskDescriptor
}
