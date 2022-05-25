package task

import (
	botharesynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/both_are_synced"
	consensuscheckpointhasprogressed "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_checkpoint_has_progressed"
	consensusishealthy "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_healthy"
	consensusissynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_synced"
	consensusissyncing "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_syncing"
	consensusisunhealthy "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/consensus_is_unhealthy"
	executionhasprogressed "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_has_progressed"
	executionishealthy "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_is_healthy"
	executionissynced "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_is_synced"
	executionisunhealthy "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/execution_is_unhealthy"
	runcommand "github.com/samcm/sync-test-coordinator/pkg/coordinator/task/run_command"
)

func AvailableTasks() MapOfRunnableInfo {
	return MapOfRunnableInfo{
		botharesynced.Name: RunnableInfo{
			Description: botharesynced.Description,
			Config:      botharesynced.DefaultConfig(),
		},
		consensuscheckpointhasprogressed.Name: RunnableInfo{
			Description: consensuscheckpointhasprogressed.Description,
			Config:      consensuscheckpointhasprogressed.DefaultConfig(),
		},
		consensusishealthy.Name: RunnableInfo{
			Description: consensusishealthy.Description,
			Config:      consensusishealthy.DefaultConfig(),
		},
		consensusissynced.Name: RunnableInfo{
			Description: consensusissynced.Description,
			Config:      consensusissynced.DefaultConfig(),
		},
		consensusissyncing.Name: RunnableInfo{
			Description: consensusissyncing.Description,
			Config:      consensusissyncing.DefaultConfig(),
		},
		consensusisunhealthy.Name: RunnableInfo{
			Description: consensusisunhealthy.Description,
			Config:      consensusisunhealthy.DefaultConfig(),
		},
		executionhasprogressed.Name: RunnableInfo{
			Description: executionhasprogressed.Description,
			Config:      executionhasprogressed.DefaultConfig(),
		},
		executionishealthy.Name: RunnableInfo{
			Description: executionishealthy.Description,
			Config:      executionishealthy.DefaultConfig(),
		},
		executionissynced.Name: RunnableInfo{
			Description: executionissynced.Description,
			Config:      executionissynced.DefaultConfig(),
		},
		executionisunhealthy.Name: RunnableInfo{
			Description: executionisunhealthy.Description,
			Config:      executionisunhealthy.DefaultConfig(),
		},
		runcommand.Name: RunnableInfo{
			Description: runcommand.Description,
			Config:      runcommand.DefaultConfig(),
		},
	}
}
