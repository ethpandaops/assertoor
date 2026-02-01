package checkclientsarehealthy

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_clients_are_healthy"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks if clients are healthy.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "goodClients",
				Type:        "array",
				Description: "Array of healthy client info objects.",
			},
			{
				Name:        "failedClients",
				Type:        "array",
				Description: "Array of unhealthy client info objects.",
			},
			{
				Name:        "totalCount",
				Type:        "int",
				Description: "Total number of clients checked.",
			},
			{
				Name:        "failedCount",
				Type:        "int",
				Description: "Number of clients that failed health check.",
			},
			{
				Name:        "goodCount",
				Type:        "int",
				Description: "Number of clients that passed health check.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

type ClientInfo struct {
	Name     string `json:"name"`
	ClRPCURL string `json:"clRpcUrl"`
	ElRPCURL string `json:"elRpcUrl"`
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
	checkCount := 0

	for {
		checkCount++

		if done, err := t.processCheck(checkCount); done {
			return err
		}

		select {
		case <-time.After(t.config.PollInterval.Duration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) processCheck(checkCount int) (bool, error) {
	expectedResult := !t.config.ExpectUnhealthy
	passResultCount := 0
	totalClientCount := 0
	goodClients := []*ClientInfo{}
	failedClients := []*ClientInfo{}
	failedClientNames := []string{}

	for _, client := range t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "") {
		totalClientCount++

		checkResult := t.processClientCheck(client)
		if checkResult == expectedResult {
			passResultCount++

			goodClients = append(goodClients, t.getClientInfo(client))

			if t.config.ExecutionRPCResultVar != "" && passResultCount == 1 && client.ExecutionClient != nil {
				t.ctx.Vars.SetVar(t.config.ExecutionRPCResultVar, client.ExecutionClient.GetEndpointConfig().URL)
			}

			if t.config.ConsensusRPCResultVar != "" && passResultCount == 1 && client.ConsensusClient != nil {
				t.ctx.Vars.SetVar(t.config.ConsensusRPCResultVar, client.ConsensusClient.GetEndpointConfig().URL)
			}
		} else {
			failedClients = append(failedClients, t.getClientInfo(client))
			failedClientNames = append(failedClientNames, client.Config.Name)
		}
	}

	requiredPassCount := t.config.MinClientCount
	if requiredPassCount == 0 {
		requiredPassCount = totalClientCount
	}

	resultPass := passResultCount >= requiredPassCount

	if goodClientsData, err := vars.GeneralizeData(goodClients); err == nil {
		t.ctx.Outputs.SetVar("goodClients", goodClientsData)
	} else {
		t.logger.Warnf("Failed setting `goodClients` output: %v", err)
	}

	if failedClientsData, err := vars.GeneralizeData(failedClients); err == nil {
		t.ctx.Outputs.SetVar("failedClients", failedClientsData)
	} else {
		t.logger.Warnf("Failed setting `failedClients` output: %v", err)
	}

	t.ctx.Outputs.SetVar("totalCount", totalClientCount)
	t.ctx.Outputs.SetVar("failedCount", totalClientCount-passResultCount)
	t.ctx.Outputs.SetVar("goodCount", passResultCount)

	t.logger.Infof("Check result: %v, Failed Clients: %v", resultPass, failedClientNames)

	switch {
	case t.config.MaxUnhealthyCount > -1 && len(failedClients) > t.config.MaxUnhealthyCount:
		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
			t.ctx.ReportProgress(0, fmt.Sprintf("Too many unhealthy clients: %d (attempt %d)", len(failedClients), checkCount))

			return true, fmt.Errorf("too many unhealthy clients: %d", len(failedClients))
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Too many unhealthy clients: %d (attempt %d)", len(failedClients), checkCount))
	case resultPass:
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, fmt.Sprintf("All clients healthy: %d/%d", passResultCount, totalClientCount))

		if !t.config.ContinueOnPass {
			return true, nil
		}
	default:
		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
			t.ctx.ReportProgress(0, fmt.Sprintf("Unhealthy clients: %v (attempt %d)", failedClientNames, checkCount))

			return true, fmt.Errorf("unhealthy clients: %v", failedClientNames)
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for healthy clients... %d/%d (attempt %d)", passResultCount, totalClientCount, checkCount))
	}

	return false, nil
}

func (t *Task) getClientInfo(client *clients.PoolClient) *ClientInfo {
	clientInfo := &ClientInfo{
		Name: client.Config.Name,
	}

	if client.ExecutionClient != nil {
		clientInfo.ElRPCURL = client.ExecutionClient.GetEndpointConfig().URL
	}

	if client.ConsensusClient != nil {
		clientInfo.ClRPCURL = client.ConsensusClient.GetEndpointConfig().URL
	}

	return clientInfo
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
