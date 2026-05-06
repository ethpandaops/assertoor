package tysmhookactivation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

const (
	apiPath        = "/tysm/v1/activations"
	requestTimeout = 30 * time.Second
)

var (
	TaskName       = "tysm_hook_activation"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Create a TTL-bound activation against a TYSM hook-control API.",
		Category:    "tysm",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "activation_id",
				Type:        "string",
				Description: "Server-assigned activation ID; pass to tysm_hook_deactivation as activation_id.",
			},
			{
				Name:        "expires_at",
				Type:        "string",
				Description: "RFC3339 timestamp at which the server-side TTL expires and the hook reverts to baseline.",
			},
			{
				Name:        "hook",
				Type:        "string",
				Description: "Name of the hook the activation targets (echoes the input).",
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

// activationRequest mirrors overlay/tysm/api.CreateActivationRequest. We
// declare it locally so this task has no dependency on the tysm overlay
// module — the two repos live in different module paths and the overlay
// only exists post-patch-apply inside a Prysm clone.
type activationRequest struct {
	Hook        string                 `json:"hook"`
	Enabled     *bool                  `json:"enabled,omitempty"`
	ConfigPatch map[string]interface{} `json:"config_patch,omitempty"`
	Duration    string                 `json:"duration"`
	Replace     bool                   `json:"replace,omitempty"`
}

// activationResponse mirrors overlay/tysm/api.ActivationView. Effective is
// intentionally decoded as a generic map: this task does not introspect it,
// it only forwards the activation handle (id, expires_at) to downstream
// tasks via outputs.
type activationResponse struct {
	ID        string                 `json:"id"`
	Hook      string                 `json:"hook"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	Effective map[string]interface{} `json:"effective"`
}

// errorResponse mirrors overlay/tysm/api.ErrorResponse. Used to surface
// the server-side reason in task error messages.
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
	t.ctx.ReportProgress(0, fmt.Sprintf("Activating hook %q", t.config.Hook))

	body, err := json.Marshal(activationRequest{
		Hook:        t.config.Hook,
		Enabled:     t.config.Enabled,
		ConfigPatch: t.config.ConfigPatch,
		Duration:    t.config.Duration.String(),
		Replace:     t.config.Replace,
	})
	if err != nil {
		return fmt.Errorf("encode activation request: %w", err)
	}

	endpoint := strings.TrimRight(t.config.Endpoint, "/") + apiPath

	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if t.config.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.config.AuthToken)
	}

	client := &http.Client{Timeout: requestTimeout}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", endpoint, err)
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

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("activation request failed: %s", formatHTTPError(resp.StatusCode, respBody))
	}

	var view activationResponse
	if err := json.Unmarshal(respBody, &view); err != nil {
		return fmt.Errorf("decode activation response: %w", err)
	}

	if view.ID == "" {
		return fmt.Errorf("server returned 201 but no activation id")
	}

	t.ctx.Outputs.SetVar("activation_id", view.ID)
	t.ctx.Outputs.SetVar("expires_at", view.ExpiresAt.Format(time.RFC3339))
	t.ctx.Outputs.SetVar("hook", view.Hook)

	t.logger.WithFields(logrus.Fields{
		"hook":          view.Hook,
		"activation_id": view.ID,
		"expires_at":    view.ExpiresAt.Format(time.RFC3339),
	}).Info("TYSM hook activation created")

	t.ctx.SetResult(types.TaskResultSuccess)
	t.ctx.ReportProgress(100, fmt.Sprintf("Activation %s expires at %s", view.ID, view.ExpiresAt.Format(time.RFC3339)))

	return nil
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
