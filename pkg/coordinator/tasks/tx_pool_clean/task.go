package txpoolclean

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/noku-team/assertoor/pkg/coordinator/clients/execution"
	"github.com/noku-team/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "tx_pool_clean"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check if the transaction pool is clean and wait until it is.",
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

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(_ context.Context) error {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	executionClients := clientPool.GetExecutionPool().GetReadyEndpoints(true)

	t.logger.Infof("Found %d execution clients", len(executionClients))

	for _, client := range executionClients {
		err := t.cleanRecursive(client)
		if err != nil {
			return err
		}
	}

	t.ctx.SetResult(types.TaskResultSuccess)

	return nil
}

func (t *Task) cleanRecursive(client *execution.Client) error {
	clean, err := isPoolClean(client)
	if err != nil {
		t.logger.Errorf("Error checking txpool: %v", err)
		t.ctx.SetResult(types.TaskResultFailure)

		return err
	}

	if clean {
		t.logger.Infof("TxPool is clean for client %s", client.GetName())
		return nil
	}

	// wait for a while before checking again
	time.Sleep(5 * time.Second)
	t.logger.Infof("TxPool is not clean for client %s, checking again...", client.GetName())

	return t.cleanRecursive(client)
}

func isPoolClean(client *execution.Client) (bool, error) {
	r, err := http.Post(client.GetEndpointConfig().URL, "application/json", strings.NewReader(
		`{"jsonrpc":"2.0","method":"txpool_content","params":[],"id":1}`,
	))

	if err != nil {
		return false, err
	}

	defer r.Body.Close()

	var resp struct {
		Result struct {
			Pending map[string]map[string]*types.PoolTransaction `json:"pending"`
		} `json:"result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return false, err
	}

	if len(resp.Result.Pending) == 0 {
		return true, nil
	}

	for _, txs := range resp.Result.Pending {
		for _, tx := range txs {
			if tx != nil {
				return false, nil
			}
		}
	}

	return true, nil
}
