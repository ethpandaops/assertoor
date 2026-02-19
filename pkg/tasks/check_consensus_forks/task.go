package checkconsensusforks

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_forks"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check for consensus layer forks.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "forks",
				Type:        "array",
				Description: "Array of fork info objects with head slot, root, and clients.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	startEpoch uint64
}

type ForkInfo struct {
	HeadSlot uint64   `json:"headSlot"`
	HeadRoot string   `json:"headRoot"`
	Clients  []string `json:"clients"`
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	blockSubscription := consensusPool.GetBlockCache().SubscribeBlockEvent(10)

	defer blockSubscription.Unsubscribe()

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		return fmt.Errorf("failed fetching wallclock: %w", err)
	}

	t.startEpoch = currentEpoch.Number()
	checkCount := 0

	for {
		select {
		case <-blockSubscription.Channel():
			checkCount++

			if done, err := t.processCheck(checkCount); done {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) processCheck(checkCount int) (bool, error) {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()
	headForks := consensusPool.GetHeadForks(t.config.MaxForkDistance)
	headForkInfo := make([]*ForkInfo, len(headForks))

	for i, headFork := range headForks {
		clients := make([]string, len(headFork.AllClients))
		for j, client := range headFork.AllClients {
			clients[j] = client.GetName()
		}

		headForkInfo[i] = &ForkInfo{
			HeadSlot: uint64(headFork.Slot),
			HeadRoot: headFork.Root.String(),
			Clients:  clients,
		}
	}

	if data, err := vars.GeneralizeData(headForkInfo); err == nil {
		t.ctx.Outputs.SetVar("forks", data)
	} else {
		t.logger.Warnf("failed setting `forks` output: %v", err)
	}

	if headForkLen := uint64(len(headForks)); headForkLen >= 1 && headForkLen-1 > t.config.MaxForkCount {
		t.logger.Warnf("check failed: too many forks. (have: %v, want <= %v)", len(headForks)-1, t.config.MaxForkCount)

		for idx, fork := range headForks {
			clients := make([]string, len(fork.AllClients))
			for _, client := range fork.AllClients {
				clients = append(clients, client.GetName())
			}

			t.logger.Infof("Fork #%v: %v [0x%x] (%v clients: [%v])", idx, fork.Slot, fork.Root, len(fork.AllClients), clients)
		}

		t.ctx.SetResult(types.TaskResultFailure)
		t.ctx.ReportProgress(0, fmt.Sprintf("Too many forks: %d (attempt %d)", len(headForks)-1, checkCount))

		return true, fmt.Errorf("too many forks: %d", len(headForks)-1)
	}

	_, currentEpoch, err := consensusPool.GetBlockCache().GetWallclock().Now()
	if err != nil {
		t.logger.Warnf("check missed: could not get current epoch from wall clock")
		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for fork check... (attempt %d)", checkCount))

		return false, nil
	}

	epochCount := currentEpoch.Number() - t.startEpoch

	if t.config.MinCheckEpochCount > 0 && epochCount < t.config.MinCheckEpochCount {
		t.logger.Warnf("Check missed: checked %v epochs, but need >= %v", epochCount, t.config.MinCheckEpochCount)
		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for fork check... %d/%d epochs (attempt %d)", epochCount, t.config.MinCheckEpochCount, checkCount))

		return false, nil
	}

	t.ctx.SetResult(types.TaskResultSuccess)
	t.ctx.ReportProgress(100, fmt.Sprintf("Fork check passed after %d epochs", epochCount))

	if !t.config.ContinueOnPass {
		return true, nil
	}

	return false, nil
}
