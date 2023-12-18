package checkclientsarehealthy

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_clients_are_healthy"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks if clients are healthy.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
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
	t.processCheck()

	for {
		select {
		case <-time.After(t.config.PollInterval.Duration):
			t.processCheck()
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *Task) processCheck() {
	expectedResult := !t.config.ExpectUnhealthy
	passResultCount := 0
	totalClientCount := 0
	failedClients := []string{}

	for _, client := range t.ctx.Scheduler.GetCoordinator().ClientPool().GetClientsByNamePatterns(t.config.ClientNamePatterns) {
		totalClientCount++

		checkResult := t.processClientCheck(client)
		if checkResult == expectedResult {
			passResultCount++
		} else {
			failedClients = append(failedClients, client.Config.Name)
		}
	}

	requiredPassCount := t.config.MinClientCount
	if requiredPassCount == 0 {
		requiredPassCount = totalClientCount
	}

	resultPass := passResultCount >= requiredPassCount

	t.logger.Infof("Check result: %v, Failed Clients: %v", resultPass, failedClients)

	if resultPass {
		t.ctx.SetResult(types.TaskResultSuccess)
	} else {
		t.ctx.SetResult(types.TaskResultNone)
	}
}

func (t *Task) processClientCheck(client *clients.PoolClient) bool {
	if !t.config.SkipConsensusCheck && client.ConsensusClient.GetStatus() == consensus.ClientStatusOffline {
		return false
	}

	if !t.config.SkipExecutionCheck && client.ExecutionClient.GetStatus() == execution.ClientStatusOffline {
		return false
	}

	return true
}
