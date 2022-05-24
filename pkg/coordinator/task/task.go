package task

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/imdario/mergo"
	"github.com/samcm/sync-test-coordinator/pkg/coordinator/helper"
	"github.com/sirupsen/logrus"
)

// Runnable represents an INDIVIDUAL task to be run. These tasks should be as small as possible.
type Runnable interface {
	Start(ctx context.Context) error
	IsComplete(ctx context.Context) (bool, error)

	Name() string
	Config() interface{}
	PollingInterval() time.Duration
	Logger() logrus.FieldLogger
}

var (
	ErrInvalidConfig = errors.New("invalid config")
)

//nolint:gocyclo // unavoidable
func NewRunnableByName(ctx context.Context, bundle *Bundle, taskName string, config *helper.RawMessage) (Runnable, error) {
	switch taskName {
	case NameSleep:
		conf := SleepConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultSleepConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewSleep(ctx, bundle, base), nil
	case NameBothAreSynced:
		conf := BothAreSyncedConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultBothAreSyncedConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewBothAreSynced(ctx, bundle, base), nil
	case NameConsensusIsHealthy:
		conf := ConsensusIsHealthyConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultConsensusIsHealthyConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewConsensusIsHealthy(ctx, bundle, base), nil

	case NameConsensusIsSynced:
		conf := ConsensusIsSyncedConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultConsensusIsSyncedConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewConsensusIsSynced(ctx, bundle, base), nil

	case NameConsensusIsSyncing:
		conf := ConsensusIsSyncedConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultConsensusIsSyncingConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewConsensusIsSyncing(ctx, bundle, base), nil

	case NameConsensusIsUnhealthy:
		conf := ConsensusIsUnhealthyConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultConsensusIsUnhealthyConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewConsensusIsUnhealthy(ctx, bundle, base), nil

	case NameConsensusCheckpointHasProgressed:
		conf := ConsensusCheckpointHasProgressedConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultConsensusCheckpointHasProgressed()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewConsensusCheckpointHasProgressed(ctx, bundle, base), nil

	case NameExecutionHasProgressed:
		conf := ExecutionHasProgressedConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultExecutionHasProgressedConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewExecutionHasProgressed(ctx, bundle, base), nil

	case NameExecutionIsHealthy:
		conf := ExecutionIsHealthyConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultExecutionIsHealthyConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewExecutionIsHealthy(ctx, bundle, base), nil

	case NameExecutionIsSynced:
		conf := ExecutionIsSyncedConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultExecutionIsSyncedConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewExecutionIsSynced(ctx, bundle, base), nil

	case NameExecutionIsUnhealthy:
		conf := ExecutionIsUnhealthyConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultExecutionIsUnhealthyConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewExecutionIsUnhealthy(ctx, bundle, base), nil

	case NameRunCommand:
		conf := RunCommandConfig{}

		if config != nil {
			if err := config.Unmarshal(&conf); err != nil {
				return nil, err
			}
		}

		base := DefaultRunCommandConfig()

		if err := mergo.Merge(&base, conf); err != nil {
			return nil, err
		}

		return NewRunCommand(ctx, bundle, base), nil
	}

	return nil, fmt.Errorf("unknown task: %s", taskName)
}
