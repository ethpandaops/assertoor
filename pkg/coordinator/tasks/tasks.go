package tasks

import (
	"github.com/erigontech/assertoor/pkg/coordinator/types"

	checkclientsarehealthy "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_clients_are_healthy"
	checkconsensusattestationstats "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_attestation_stats"
	checkconsensusblockproposals "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_block_proposals"
	checkconsensusfinality "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_finality"
	checkconsensusforks "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_forks"
	checkconsensusproposerduty "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_proposer_duty"
	checkconsensusreorgs "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_reorgs"
	checkconsensusslotrange "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_slot_range"
	checkconsensussyncstatus "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_sync_status"
	checkconsensusvalidatorstatus "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_consensus_validator_status"
	checkethcall "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_eth_call"
	checkexecutionsyncstatus "github.com/erigontech/assertoor/pkg/coordinator/tasks/check_execution_sync_status"
	generateblobtransactions "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_blob_transactions"
	generateblschanges "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_bls_changes"
	generatechildwallet "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_child_wallet"
	generateconsolidations "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_consolidations"
	generatedeposits "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_deposits"
	generateeoatransactions "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_eoa_transactions"
	generateexits "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_exits"
	generateslashings "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_slashings"
	generatetransaction "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_transaction"
	generatewithdrawalrequests "github.com/erigontech/assertoor/pkg/coordinator/tasks/generate_withdrawal_requests"
	getconsensusspecs "github.com/erigontech/assertoor/pkg/coordinator/tasks/get_consensus_specs"
	checkexecutionblock "github.com/erigontech/assertoor/pkg/coordinator/tasks/get_execution_block"
	getpubkeysfrommnemonic "github.com/erigontech/assertoor/pkg/coordinator/tasks/get_pubkeys_from_mnemonic"
	getrandommnemonic "github.com/erigontech/assertoor/pkg/coordinator/tasks/get_random_mnemonic"
	getwalletdetails "github.com/erigontech/assertoor/pkg/coordinator/tasks/get_wallet_details"
	runcommand "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_command"
	runexternaltasks "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_external_tasks"
	runshell "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_shell"
	runtaskbackground "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_task_background"
	runtaskmatrix "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_task_matrix"
	runtaskoptions "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_task_options"
	runtasks "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_tasks"
	runtasksconcurrent "github.com/erigontech/assertoor/pkg/coordinator/tasks/run_tasks_concurrent"
	sleep "github.com/erigontech/assertoor/pkg/coordinator/tasks/sleep"
	txpoolclean "github.com/erigontech/assertoor/pkg/coordinator/tasks/tx_pool_clean"
	txpoollatencyanalysis "github.com/erigontech/assertoor/pkg/coordinator/tasks/tx_pool_latency_analysis"
	txpoolthroughputanalysis "github.com/erigontech/assertoor/pkg/coordinator/tasks/tx_pool_throughput_analysis"
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
	txpoollatencyanalysis.TaskDescriptor,
	txpoolthroughputanalysis.TaskDescriptor,
	txpoolclean.TaskDescriptor,
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
