package checkclientsarehealthy

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
	"github.com/ethpandaops/minccino/pkg/coordinator/clients/consensus"
	"github.com/ethpandaops/minccino/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/minccino/pkg/coordinator/types"
	"github.com/imdario/mergo"
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
	config := DefaultConfig()

	if options.Config != nil {
		conf := &Config{}
		if err := options.Config.Unmarshal(&conf); err != nil {
			return nil, fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}

		if err := mergo.Merge(&config, conf, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("error merging task config for %v: %w", TaskName, err)
		}
	}

	return &Task{
		ctx:     ctx,
		options: options,
		config:  config,
		logger:  ctx.Scheduler.GetLogger().WithField("task", TaskName),
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.options.Title
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

func (t *Task) ValidateConfig() error {
	if err := t.config.Validate(); err != nil {
		return err
	}

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
		if checkResult != expectedResult {
			passResultCount++

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
