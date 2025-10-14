package checkethconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_eth_config"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks that all execution clients return the same eth_config (EIP-7910)",
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
	// Get the client pool from the scheduler
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	// Get matching clients from the pool
	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints(true)
		if len(clients) == 0 {
			t.logger.Error("check failed: no matching clients found")
			t.ctx.SetResult(types.TaskResultFailure)

			return nil
		}
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			t.logger.Errorf("check failed: no matching clients found with pattern %v", t.config.ClientPattern)
			t.ctx.SetResult(types.TaskResultFailure)

			return nil
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	// Query eth_config from all clients
	type clientResult struct {
		client     *execution.Client
		configJSON string
		err        error
	}

	results := make([]clientResult, len(clients))

	for i, client := range clients {
		t.logger.Infof("querying eth_config from client %v", client.GetName())

		ethConfig, err := client.GetRPCClient().GetEthConfig(ctx)
		if err != nil {
			results[i] = clientResult{
				client: client,
				err:    err,
			}

			t.logger.WithField("client", client.GetName()).Errorf("RPC error when querying eth_config: %v", err)

			if t.config.FailOnMismatch {
				t.ctx.SetResult(types.TaskResultFailure)
				return nil
			}

			continue
		}

		// Convert to JSON for comparison
		configBytes, err := json.MarshalIndent(ethConfig, "", "  ")
		if err != nil {
			results[i] = clientResult{
				client: client,
				err:    fmt.Errorf("failed to marshal config: %w", err),
			}

			t.logger.WithField("client", client.GetName()).Errorf("error marshaling eth_config: %v", err)

			if t.config.FailOnMismatch {
				t.ctx.SetResult(types.TaskResultFailure)
				return nil
			}

			continue
		}

		results[i] = clientResult{
			client:     client,
			configJSON: string(configBytes),
		}

		t.logger.WithField("client", client.GetName()).Debugf("eth_config response:\n%s", string(configBytes))
		t.logger.Infof("client %v returned eth_config successfully", client.GetName())
	}

	// Check for consistency
	var referenceConfig string

	mismatchFound := false
	configMap := make(map[string][]string) // config JSON -> list of client names

	for _, result := range results {
		if result.err != nil {
			continue
		}

		if referenceConfig == "" {
			referenceConfig = result.configJSON
			t.ctx.Outputs.SetVar("ethConfig", result.configJSON)
		}

		// Track which clients returned which config
		configMap[result.configJSON] = append(configMap[result.configJSON], result.client.GetName())

		if result.configJSON != referenceConfig {
			mismatchFound = true
		}
	}

	if mismatchFound {
		// Build diff output
		var diffBuilder strings.Builder

		diffBuilder.WriteString("eth_config mismatch detected across clients:\n\n")

		configIndex := 1
		for config, clientNames := range configMap {
			diffBuilder.WriteString(fmt.Sprintf("Config variant #%d (clients: %v):\n", configIndex, clientNames))
			diffBuilder.WriteString(config)
			diffBuilder.WriteString("\n\n")

			configIndex++
		}

		t.logger.Error(diffBuilder.String())

		if t.config.FailOnMismatch {
			t.ctx.SetResult(types.TaskResultFailure)
		} else {
			t.ctx.SetResult(types.TaskResultNone)
		}

		return nil
	}

	// All checks passed
	if len(results) > 0 {
		t.logger.Infof("all %d clients returned consistent eth_config", len(results))
		t.logger.Debugf("consistent eth_config:\n%s", referenceConfig)
		t.ctx.SetResult(types.TaskResultSuccess)
	}

	return nil
}
