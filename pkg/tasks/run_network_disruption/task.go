package runnetworkdisruption

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

const (
	// maxResponseSize caps disruptoor API response bodies; state documents
	// are tiny, so 10MB is a generous safety bound.
	maxResponseSize = 10 * 1024 * 1024

	// maxUpdateAttempts bounds the read-merge-write retry loop when a
	// concurrent writer invalidates our ETag between GET and PUT.
	maxUpdateAttempts = 3

	stateKeyPartitions = "partitions"
	stateKeyIsolations = "isolations"
	stateKeyShaping    = "shaping"

	outputTypeInt = "int"
)

// Compile-time interface compliance check.
var _ types.Task = (*Task)(nil)

var (
	TaskName       = "run_network_disruption"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Applies or heals network disruptions (partitions, isolations, shaping) via the disruptoor API.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "appliedState",
				Type:        "object",
				Description: "The disruptoor state applied after the action (reflects reality, fetched via GET /v1/state).",
			},
			{
				Name:        "partitionCount",
				Type:        outputTypeInt,
				Description: "Number of active partitions after the action.",
			},
			{
				Name:        "isolationCount",
				Type:        outputTypeInt,
				Description: "Number of active isolations after the action.",
			},
			{
				Name:        "shapingCount",
				Type:        outputTypeInt,
				Description: "Number of active shaping rules after the action.",
			},
		},
		NewTask: NewTask,
	}

	errStalePut = fmt.Errorf("state changed between read and write")
)

// Task implements the run_network_disruption task.
type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	httpClient *http.Client
}

// NewTask creates a new run_network_disruption task.
func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

// Config returns the task configuration.
func (t *Task) Config() interface{} {
	return t.config
}

// Timeout returns the task timeout.
func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

// LoadConfig loads and validates the task configuration.
func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// Parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// Load dynamic vars
	if err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars); err != nil {
		return err
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	t.httpClient = &http.Client{
		Timeout: config.RequestTimeout.Duration,
	}

	return nil
}

// Execute runs the task.
func (t *Task) Execute(ctx context.Context) error {
	if t.config.AwaitAPITimeout.Duration > 0 {
		t.ctx.ReportProgress(0, "Waiting for disruptoor API...")

		if err := t.awaitHealthy(ctx); err != nil {
			return err
		}
	}

	t.ctx.ReportProgress(25, fmt.Sprintf("Performing %v action...", t.config.Action))

	var err error

	switch t.config.Action {
	case ActionClear:
		err = t.clearState(ctx)
	case ActionSet:
		err = t.putState(ctx, t.buildDesiredState(), "")
	case ActionUpdate:
		err = t.updateState(ctx)
	}

	if err != nil {
		return err
	}

	// GET reflects applied reality per the disruptoor API contract, so use
	// it (rather than the PUT echo) as the source for the task outputs.
	applied, _, err := t.getState(ctx)
	if err != nil {
		return fmt.Errorf("fetching applied state: %w", err)
	}

	t.setOutputs(applied)
	t.ctx.ReportProgress(100, fmt.Sprintf("Disruption %v action completed", t.config.Action))
	t.logger.Infof("disruptoor %v action applied (%d partitions, %d isolations, %d shaping rules active)",
		t.config.Action, entryCount(applied, stateKeyPartitions),
		entryCount(applied, stateKeyIsolations), entryCount(applied, stateKeyShaping))

	return nil
}

// awaitHealthy polls the disruptoor healthz endpoint until it responds OK or
// the configured timeout elapses.
func (t *Task) awaitHealthy(ctx context.Context) error {
	deadline := time.Now().Add(t.config.AwaitAPITimeout.Duration)

	for {
		err := t.probeHealth(ctx)
		if err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("disruptoor API not healthy after %v: %w", t.config.AwaitAPITimeout.Duration, err)
		}

		t.logger.Debugf("disruptoor API not healthy yet: %v", err)

		select {
		case <-time.After(t.config.PollInterval.Duration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (t *Task) probeHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.config.DisruptoorURL+"/v1/healthz", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthz returned status %d", resp.StatusCode)
	}

	return nil
}

// buildDesiredState assembles the wire-format state document from the
// configured passthrough entries.
func (t *Task) buildDesiredState() map[string]any {
	state := make(map[string]any, 3)
	if len(t.config.Partitions) > 0 {
		state[stateKeyPartitions] = t.config.Partitions
	}

	if len(t.config.Isolations) > 0 {
		state[stateKeyIsolations] = t.config.Isolations
	}

	if len(t.config.Shaping) > 0 {
		state[stateKeyShaping] = t.config.Shaping
	}

	return state
}

// updateState merges the configured entries into the current disruptoor
// state: entries named in removeNames are dropped, then each configured
// entry replaces its same-name predecessor or is appended. The write is
// guarded with If-Match and retried when a concurrent writer wins the race.
func (t *Task) updateState(ctx context.Context) error {
	for attempt := 1; ; attempt++ {
		current, etag, err := t.getState(ctx)
		if err != nil {
			return fmt.Errorf("fetching current state: %w", err)
		}

		merged := t.mergeState(current)

		err = t.putState(ctx, merged, etag)
		if err == nil {
			return nil
		}

		if err != errStalePut || attempt >= maxUpdateAttempts {
			return err
		}

		t.logger.Warnf("disruptoor state changed concurrently, retrying update (attempt %d/%d)", attempt, maxUpdateAttempts)
	}
}

func (t *Task) mergeState(current map[string]any) map[string]any {
	removeSet := make(map[string]bool, len(t.config.RemoveNames))
	for _, name := range t.config.RemoveNames {
		removeSet[name] = true
	}

	merged := make(map[string]any, 3)

	for _, key := range []string{stateKeyPartitions, stateKeyIsolations, stateKeyShaping} {
		entries := stateEntries(current, key)
		kept := make([]map[string]any, 0, len(entries))

		for _, entry := range entries {
			if name, ok := entry["name"].(string); ok && removeSet[name] {
				continue
			}

			kept = append(kept, entry)
		}

		if len(kept) > 0 {
			merged[key] = kept
		}
	}

	mergeEntries(merged, stateKeyPartitions, t.config.Partitions)
	mergeEntries(merged, stateKeyIsolations, t.config.Isolations)
	mergeEntries(merged, stateKeyShaping, t.config.Shaping)

	return merged
}

func (t *Task) getState(ctx context.Context) (state map[string]any, etag string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.config.DisruptoorURL+"/v1/state", http.NoBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", apiError("GET /v1/state", resp.StatusCode, body)
	}

	state = make(map[string]any, 3)
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, "", fmt.Errorf("failed to parse state response: %w", err)
	}

	return state, resp.Header.Get("ETag"), nil
}

// putState PUTs the desired state, optionally guarded by an If-Match ETag.
// Returns errStalePut on a 412 so callers can re-read and retry.
func (t *Task) putState(ctx context.Context, state map[string]any, etag string) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, t.config.DisruptoorURL+"/v1/state", bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if etag != "" {
		req.Header.Set("If-Match", etag)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusPreconditionFailed {
		return errStalePut
	}

	if resp.StatusCode != http.StatusOK {
		return apiError("PUT /v1/state", resp.StatusCode, body)
	}

	return nil
}

func (t *Task) clearState(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.config.DisruptoorURL+"/v1/state/clear", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	body, err := readBody(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return apiError("POST /v1/state/clear", resp.StatusCode, body)
	}

	return nil
}

func (t *Task) setOutputs(applied map[string]any) {
	if data, err := vars.GeneralizeData(applied); err != nil {
		t.logger.Warnf("failed to generalize appliedState: %v", err)
	} else {
		t.ctx.Outputs.SetVar("appliedState", data)
	}

	t.ctx.Outputs.SetVar("partitionCount", entryCount(applied, stateKeyPartitions))
	t.ctx.Outputs.SetVar("isolationCount", entryCount(applied, stateKeyIsolations))
	t.ctx.Outputs.SetVar("shapingCount", entryCount(applied, stateKeyShaping))
}

// mergeEntries replaces same-name entries in dst[key] with their configured
// counterparts and appends entries that aren't present yet.
func mergeEntries(dst map[string]any, key string, entries []map[string]any) {
	if len(entries) == 0 {
		return
	}

	existing, _ := dst[key].([]map[string]any)

	for _, entry := range entries {
		name, _ := entry["name"].(string)
		replaced := false

		for i, cur := range existing {
			if curName, ok := cur["name"].(string); ok && curName == name {
				existing[i] = entry
				replaced = true

				break
			}
		}

		if !replaced {
			existing = append(existing, entry)
		}
	}

	dst[key] = existing
}

// stateEntries extracts a list of entry objects from a decoded state
// document, tolerating both absent keys and unexpected shapes.
func stateEntries(state map[string]any, key string) []map[string]any {
	raw, ok := state[key].([]any)
	if !ok {
		return nil
	}

	out := make([]map[string]any, 0, len(raw))

	for _, item := range raw {
		if entry, ok := item.(map[string]any); ok {
			out = append(out, entry)
		}
	}

	return out
}

func entryCount(state map[string]any, key string) int {
	switch entries := state[key].(type) {
	case []any:
		return len(entries)
	case []map[string]any:
		return len(entries)
	default:
		return 0
	}
}

func readBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) > maxResponseSize {
		return nil, fmt.Errorf("response body exceeds max size (%d bytes)", maxResponseSize)
	}

	return body, nil
}

// apiError surfaces the server-side error message when disruptoor rejects a
// request (its validation errors are precise and actionable).
func apiError(op string, status int, body []byte) error {
	var errBody struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &errBody); err == nil && errBody.Error != "" {
		return fmt.Errorf("%s returned status %d: %s", op, status, errBody.Error)
	}

	return fmt.Errorf("%s returned status %d", op, status)
}
