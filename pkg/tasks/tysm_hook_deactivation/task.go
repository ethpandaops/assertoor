package tysmhookdeactivation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

const (
	apiPath        = "/tysm/v1/activations/"
	requestTimeout = 30 * time.Second
)

var (
	TaskName       = "tysm_hook_deactivation"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Cancel a TYSM hook activation. Designed to be placed in a test's cleanupTasks block.",
		Category:    "tysm",
		Config:      DefaultConfig(),
		Outputs:     []types.TaskOutputDefinition{},
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

// errorResponse mirrors overlay/tysm/api.ErrorResponse — declared locally
// so this task does not depend on the tysm overlay module (different module
// path, only exists post-patch-apply).
type errorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
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

func (t *Task) Execute(ctx context.Context) error {
	t.ctx.ReportProgress(0, fmt.Sprintf("Deactivating activation %q", t.config.ActivationID))

	endpoint := strings.TrimRight(t.config.Endpoint, "/") + apiPath + url.PathEscape(t.config.ActivationID)

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodDelete, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	if t.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.AuthToken)
	}

	client := &http.Client{Timeout: requestTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %w", endpoint, err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			t.logger.WithError(closeErr).Warn("failed to close response body")
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusNoContent:
		t.logger.WithField("activation_id", t.config.ActivationID).Info("TYSM hook activation cancelled")
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, "Activation cancelled")

		return nil

	case resp.StatusCode == http.StatusNotFound && t.config.IgnoreNotFound:
		// TTL may have already fired between the activation task running
		// and the cleanup task running. Treat that as success — the
		// invariant the caller wants (no activation in force) holds.
		t.logger.WithField("activation_id", t.config.ActivationID).Info("TYSM hook activation already gone (404); treating as cancelled")
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, "Activation already expired")

		return nil

	default:
		return fmt.Errorf("deactivation request failed: %s", formatHTTPError(resp.StatusCode, respBody))
	}
}

// formatHTTPError flattens the server's JSON error envelope into one line
// suitable for an Execute error return. Falls back to the raw body when
// the response is not valid JSON or carries no Error field.
func formatHTTPError(status int, body []byte) string {
	var errResp errorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		if errResp.Details != "" {
			return fmt.Sprintf("HTTP %d: %s (%s)", status, errResp.Error, errResp.Details)
		}

		return fmt.Sprintf("HTTP %d: %s", status, errResp.Error)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Sprintf("HTTP %d", status)
	}

	return fmt.Sprintf("HTTP %d: %s", status, trimmed)
}
