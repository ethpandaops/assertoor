package checkexecutionconversionstate

import (
	"context"
	"fmt"
	"time"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution/rpc"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_execution_conversion_state"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks execution clients for their verkle conversion status.",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx         *types.TaskContext
	options     *types.TaskOptions
	config      Config
	logger      logrus.FieldLogger
	firstHeight map[uint16]uint64
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:         ctx,
		options:     options,
		logger:      ctx.Logger.GetLogger(),
		firstHeight: map[uint16]uint64{},
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
	t.processCheck(ctx)

	for {
		select {
		case <-time.After(t.config.PollInterval.Duration):
			t.processCheck(ctx)
		case <-ctx.Done():
			return nil
		}
	}
}

func (t *Task) processCheck(ctx context.Context) {
	allResultsPass := true
	failedClients := []string{}

	for _, client := range t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(t.config.ClientPattern, "") {
		var checkResult bool

		checkLogger := t.logger.WithField("client", client.Config.Name)
		conversionStatus, err := client.ExecutionClient.GetRPCClient().GetVerkleConversionState(ctx)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			checkLogger.Warnf("error fetching verkle conversion status: %v", err)

			checkResult = false
		} else {
			checkResult = t.processClientCheck(conversionStatus, checkLogger)
		}

		if !checkResult {
			allResultsPass = false

			failedClients = append(failedClients, client.Config.Name)
		}
	}

	t.logger.Infof("Check result: %v, Failed Clients: %v", allResultsPass, failedClients)

	switch {
	case allResultsPass:
		t.ctx.SetResult(types.TaskResultSuccess)
	case t.config.FailOnUnexpected:
		t.ctx.SetResult(types.TaskResultFailure)
	default:
		t.ctx.SetResult(types.TaskResultNone)
	}
}

func (t *Task) processClientCheck(conversionStatus *rpc.VerkleConversionState, checkLogger logrus.FieldLogger) bool {
	if conversionStatus.Started != t.config.ExpectStarted {
		checkLogger.Debugf("check failed. check: ExpectStarted, expected: %v, got: %v", t.config.ExpectStarted, conversionStatus.Started)
		return false
	}

	if conversionStatus.Ended != t.config.ExpectFinished {
		checkLogger.Debugf("check failed. check: ExpectFinished, expected: %v, got: %v", t.config.ExpectFinished, conversionStatus.Ended)
		return false
	}

	return true
}
