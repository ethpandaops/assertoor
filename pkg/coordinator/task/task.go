package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/helper"
	botharesynced "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/both_are_synced"
	botharesyncing "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/both_are_syncing"
	consensuscheckpointhasprogressed "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/consensus_checkpoint_has_progressed"
	consensusishealthy "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/consensus_is_healthy"
	consensusissynced "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/consensus_is_synced"
	consensusissyncing "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/consensus_is_syncing"
	consensusisunhealthy "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/consensus_is_unhealthy"
	executionhasprogressed "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/execution_has_progressed"
	executionishealthy "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/execution_is_healthy"
	executionissynced "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/execution_is_synced"
	executionissyncing "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/execution_is_syncing"
	executionisunhealthy "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/execution_is_unhealthy"
	runcommand "github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/run_command"
	"github.com/ethpandaops/sync-test-coordinator/pkg/coordinator/task/sleep"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
)

// Runnable represents an INDIVIDUAL task to be run. These tasks should be as small as possible.
type Runnable interface {
	Start(ctx context.Context) error
	IsComplete(ctx context.Context) (bool, error)
	ValidateConfig() error

	Description() string
	Name() string
	Title() string
	Config() interface{}
	PollingInterval() time.Duration
	Logger() logrus.FieldLogger
	Timeout() time.Duration
}

var (
	ErrInvalidConfig = errors.New("invalid config")
)

//nolint:gocyclo // unavoidable
func NewRunnableByName(ctx context.Context, log logrus.FieldLogger, executionURL, consensusURL, taskName string, config *helper.RawMessage, title string, timeout time.Duration) (Runnable, error) {
	switch taskName {
	case sleep.Name:
		conf := sleep.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := sleep.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return sleep.NewTask(ctx, log, base, title, timeout), nil
	case botharesynced.Name:
		conf := botharesynced.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := botharesynced.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return botharesynced.NewTask(ctx, log, consensusURL, executionURL, base, title, timeout), nil

	case botharesyncing.Name:
		conf := botharesyncing.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := botharesyncing.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return botharesyncing.NewTask(ctx, log, consensusURL, executionURL, base, title, timeout), nil

	case consensusishealthy.Name:
		conf := consensusishealthy.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := consensusishealthy.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return consensusishealthy.NewTask(ctx, log, consensusURL, base, title, timeout), nil

	case consensusissynced.Name:
		conf := consensusissynced.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := consensusissynced.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return consensusissynced.NewTask(ctx, log, consensusURL, base, title, timeout), nil

	case consensusissyncing.Name:
		conf := consensusissyncing.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := consensusissyncing.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return consensusissyncing.NewTask(ctx, log, consensusURL, base, title, timeout), nil

	case consensusisunhealthy.Name:
		conf := consensusisunhealthy.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := consensusisunhealthy.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return consensusisunhealthy.NewTask(ctx, log, consensusURL, base, title, timeout), nil

	case consensuscheckpointhasprogressed.Name:
		conf := consensuscheckpointhasprogressed.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := consensuscheckpointhasprogressed.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return consensuscheckpointhasprogressed.NewTask(ctx, log, consensusURL, base, title, timeout), nil

	case executionhasprogressed.Name:
		conf := executionhasprogressed.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := executionhasprogressed.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return executionhasprogressed.NewTask(ctx, log, executionURL, base, title, timeout), nil

	case executionishealthy.Name:
		conf := executionishealthy.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := executionishealthy.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return executionishealthy.NewTask(ctx, log, executionURL, base, title, timeout), nil

	case executionissynced.Name:
		conf := executionissynced.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := executionissynced.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return executionissynced.NewTask(ctx, log, executionURL, base, title, timeout), nil

	case executionissyncing.Name:
		conf := executionissyncing.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := executionissyncing.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return executionissyncing.NewTask(ctx, log, executionURL, base, title, timeout), nil

	case executionisunhealthy.Name:
		conf := executionisunhealthy.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := executionisunhealthy.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return executionisunhealthy.NewTask(ctx, log, executionURL, base, title, timeout), nil

	case runcommand.Name:
		conf := runcommand.Config{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := runcommand.DefaultConfig()

		if err := mergo.Merge(&base, conf, mergo.WithOverride); err != nil {
			return nil, err
		}

		return runcommand.NewTask(ctx, log, base, title, timeout), nil
	}

	return nil, fmt.Errorf("unknown task: %s", taskName)
}
