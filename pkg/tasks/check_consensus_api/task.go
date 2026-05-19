package checkconsensusapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/sirupsen/logrus"
)

// recentSlotOffset is how many slots back from head the {recent_*}
// placeholders resolve to. 4 slots ≈ 48s with a 12s slot, which is
// well past the point where derived data like execution-payload
// envelopes should be available.
const recentSlotOffset = 4

const (
	defaultRequestTimeout    = 30 * time.Second
	defaultOverallTimeout    = 90 * time.Second
	defaultSSETimeoutSeconds = 36

	// Per-client result classifications
	resultPass    = "pass"
	resultPartial = "partial"
	resultFail    = "fail"
	resultSkipped = "skipped"

	// Common literals reused across helpers.
	methodGet      = "GET"
	clientTypeUnk  = "unknown"
	maxResponseLen = 256 * 1024
)

var (
	TaskName       = "check_consensus_api"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Probe a single beacon-API endpoint or SSE topic across all consensus clients and classify each response against the spec.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "results",
				Type:        "array",
				Description: "Per-client check results.",
			},
			{
				Name:        "matrixRow",
				Type:        "object",
				Description: "Compatibility matrix row keyed by client-type (lighthouse, teku, prysm, grandine, nimbus, lodestar).",
			},
			{
				Name:        "passCount",
				Type:        "int",
				Description: "Number of clients with status 'pass'.",
			},
			{
				Name:        "partialCount",
				Type:        "int",
				Description: "Number of clients with status 'partial'.",
			},
			{
				Name:        "failCount",
				Type:        "int",
				Description: "Number of clients with status 'fail'.",
			},
			{
				Name:        "skippedCount",
				Type:        "int",
				Description: "Number of clients that were skipped.",
			},
			{
				Name:        "totalCount",
				Type:        "int",
				Description: "Total number of clients evaluated.",
			},
			{
				Name:        "rowId",
				Type:        "string",
				Description: "The rowId from the config (echoed for aggregator use).",
			},
			{
				Name:        "rowTitle",
				Type:        "string",
				Description: "The rowTitle from the config (echoed for aggregator use).",
			},
			{
				Name:        "referenceUrl",
				Type:        "string",
				Description: "The reference URL from the config (echoed for aggregator use).",
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

// PerClientResult captures the per-client outcome of one endpoint probe.
type PerClientResult struct {
	Client       string   `json:"client"`
	ClientType   string   `json:"clientType"`
	Status       string   `json:"status"` // pass | partial | fail | skipped
	HTTPStatus   int      `json:"httpStatus,omitempty"`
	URL          string   `json:"url,omitempty"`
	DurationMs   int64    `json:"durationMs"`
	Note         string   `json:"note,omitempty"`
	Error        string   `json:"error,omitempty"`
	SchemaErrors []string `json:"schemaErrors,omitempty"`
	EventCount   int      `json:"eventCount,omitempty"`
}

// MatrixCell is one cell of the aggregated client-type-level matrix row.
type MatrixCell struct {
	Result     string `json:"result"` // pass | partial | fail | skipped | absent
	Note       string `json:"note,omitempty"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
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

	if err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars); err != nil {
		return err
	}

	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	// Compile schemas once, up front.
	successSchema, err := compileSchema(t.config.ResponseSchema)
	if err != nil {
		return fmt.Errorf("invalid responseSchema: %w", err)
	}

	errorSchema, err := compileSchema(t.config.ErrorSchema)
	if err != nil {
		return fmt.Errorf("invalid errorSchema: %w", err)
	}

	eventSchema, err := compileSchema(t.config.EventSchema)
	if err != nil {
		return fmt.Errorf("invalid eventSchema: %w", err)
	}

	// Collect clients to probe.
	allClients := t.ctx.Scheduler.GetServices().ClientPool().GetClientsByNamePatterns(
		t.config.ClientPattern, t.config.ExcludeClientPattern,
	)
	if len(allClients) == 0 {
		t.logger.Warn("no consensus clients matched the configured patterns")
	}

	// Resolve placeholders from chain state where possible.
	resolvedPath, resolvedParams, placeholderErr := t.resolvePath(ctx, allClients)
	if placeholderErr != nil {
		t.logger.WithError(placeholderErr).Warn("failed to resolve path placeholders from chain state; continuing with explicit pathParams only")
	}

	// Apply per-client overrides for {builder_index} etc. captured at resolve time.
	t.logger.Infof("checking endpoint: %s %s (clients: %d)", t.cfgMethod(), resolvedPath, len(allClients))

	overallCtx, cancel := context.WithTimeout(ctx, t.cfgOverallTimeout())
	defer cancel()

	sem := make(chan struct{}, t.cfgConcurrency())
	results := make([]*PerClientResult, len(allClients))
	wg := sync.WaitGroup{}

	for i, client := range allClients {
		i, client := i, client

		wg.Add(1)

		go func() {
			defer wg.Done()

			sem <- struct{}{}

			defer func() { <-sem }()

			r := t.checkClient(overallCtx, client, resolvedPath, resolvedParams, successSchema, errorSchema, eventSchema)
			results[i] = r

			t.logger.WithFields(logrus.Fields{
				"client":  r.Client,
				"type":    r.ClientType,
				"status":  r.Status,
				"http":    r.HTTPStatus,
				"durMs":   r.DurationMs,
				"events":  r.EventCount,
				"note":    r.Note,
				"errMsg":  r.Error,
				"schemaE": len(r.SchemaErrors),
			}).Info("client check complete")
		}()
	}

	wg.Wait()

	// Sort results by client name for deterministic outputs.
	sort.Slice(results, func(a, b int) bool {
		if results[a] == nil {
			return false
		}

		if results[b] == nil {
			return true
		}

		return results[a].Client < results[b].Client
	})

	passCount, partialCount, failCount, skipCount := 0, 0, 0, 0

	for _, r := range results {
		if r == nil {
			continue
		}

		switch r.Status {
		case resultPass:
			passCount++
		case resultPartial:
			partialCount++
		case resultFail:
			failCount++
		case resultSkipped:
			skipCount++
		}
	}

	// Build matrixRow: collapse multiple clients of the same type with
	// worst-case status (fail > partial > pass > skipped).
	matrixRow := buildMatrixRow(results)

	// Emit outputs.
	if resultsData, err := vars.GeneralizeData(results); err == nil {
		t.ctx.Outputs.SetVar("results", resultsData)
	}

	if rowData, err := vars.GeneralizeData(matrixRow); err == nil {
		t.ctx.Outputs.SetVar("matrixRow", rowData)
	}

	t.ctx.Outputs.SetVar("passCount", passCount)
	t.ctx.Outputs.SetVar("partialCount", partialCount)
	t.ctx.Outputs.SetVar("failCount", failCount)
	t.ctx.Outputs.SetVar("skippedCount", skipCount)
	t.ctx.Outputs.SetVar("totalCount", len(results))
	t.ctx.Outputs.SetVar("rowId", t.config.RowID)
	t.ctx.Outputs.SetVar("rowTitle", t.config.RowTitle)
	t.ctx.Outputs.SetVar("referenceUrl", t.config.ReferenceURL)

	t.logger.Infof("[%s] results: %d pass, %d partial, %d fail, %d skipped (total %d)",
		t.config.RowID, passCount, partialCount, failCount, skipCount, len(results))

	switch {
	case t.config.FailOnAnyError && (failCount+partialCount) > 0:
		t.ctx.SetResult(types.TaskResultFailure)

		return fmt.Errorf("[%s] not all clients passed (fail=%d, partial=%d)", t.config.RowID, failCount, partialCount)
	case t.config.FailOnAllError && passCount == 0:
		t.ctx.SetResult(types.TaskResultFailure)

		return fmt.Errorf("[%s] no client passed", t.config.RowID)
	default:
		t.ctx.SetResult(types.TaskResultSuccess)
	}

	return nil
}

func (t *Task) cfgMethod() string {
	if t.config.SSE != nil {
		return "SSE"
	}

	if t.config.Method == "" {
		return methodGet
	}

	return strings.ToUpper(t.config.Method)
}

func (t *Task) cfgRequestTimeout() time.Duration {
	if t.config.RequestTimeout.Duration > 0 {
		return t.config.RequestTimeout.Duration
	}

	return defaultRequestTimeout
}

func (t *Task) cfgOverallTimeout() time.Duration {
	if t.config.OverallTimeout.Duration > 0 {
		return t.config.OverallTimeout.Duration
	}

	return defaultOverallTimeout
}

func (t *Task) cfgConcurrency() int {
	if t.config.Concurrency > 0 {
		return t.config.Concurrency
	}

	return 6
}

// resolvePath fills in {placeholders} from explicit pathParams and from chain
// state where possible. Returns the templated path plus the final
// placeholder->value map for diagnostics.
func (t *Task) resolvePath(ctx context.Context, allClients []*clients.PoolClient) (finalPath string, params map[string]string, err error) {
	pathTpl := t.config.Path
	params = map[string]string{}

	// Start with explicit pathParams.
	for k, v := range t.config.PathParams {
		params[k] = v
	}

	// Identify placeholders still missing.
	placeholderRE := regexp.MustCompile(`\{([a-zA-Z0-9_+\-]+)\}`)

	missing := []string{}

	for _, m := range placeholderRE.FindAllStringSubmatch(pathTpl, -1) {
		key := m[1]
		if _, ok := params[key]; !ok {
			missing = append(missing, key)
		}
	}

	if len(missing) == 0 {
		return pathTpl, params, nil
	}

	// Need chain state. Use the first ready client we can find.
	var refClient *clients.PoolClient

	for _, c := range allClients {
		if c.ConsensusClient != nil && c.ConsensusClient.GetStatus() == consensus.ClientStatusOnline {
			refClient = c
			break
		}
	}

	if refClient == nil && len(allClients) > 0 {
		refClient = allClients[0]
	}

	pctx := pathContext{slotsPerEpoch: 32}

	if refClient != nil && refClient.ConsensusClient != nil {
		ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		rpc := refClient.ConsensusClient.GetRPCClient()
		if header, herr := rpc.GetLatestBlockHead(ctx2); herr == nil && header != nil {
			pctx.headSlot = uint64(header.Header.Message.Slot)
			pctx.headRoot = fmt.Sprintf("%#x", header.Root)
		} else {
			err = herr
		}

		if specs, serr := rpc.GetConfigSpecs(ctx2); serr == nil {
			if v, ok := specs["SLOTS_PER_EPOCH"]; ok {
				if vu, ok := v.(uint64); ok && vu > 0 {
					pctx.slotsPerEpoch = vu
				}
			}
		}

		if pctx.slotsPerEpoch > 0 {
			pctx.headEpoch = pctx.headSlot / pctx.slotsPerEpoch
		}

		// Compute a "recent" reference point a few slots back so
		// derived data (execution-payload envelopes, attestations,
		// etc.) is reliably available across all probed clients.
		// Defaults to head when chain is younger than the offset.
		if pctx.headSlot > recentSlotOffset {
			pctx.recentSlot = pctx.headSlot - recentSlotOffset
		} else {
			pctx.recentSlot = pctx.headSlot
		}

		if pctx.slotsPerEpoch > 0 {
			pctx.recentEpoch = pctx.recentSlot / pctx.slotsPerEpoch
		}

		// Fetch the canonical block root at that slot. Best-effort:
		// fall back to the head root if the lookup fails.
		ctx3, cancel3 := context.WithTimeout(ctx, 5*time.Second)
		if hdr, herr := rpc.GetBlockHeaderBySlot(ctx3, phase0.Slot(pctx.recentSlot)); herr == nil && hdr != nil {
			pctx.recentRoot = fmt.Sprintf("%#x", hdr.Root)
		}

		cancel3()

		if pctx.recentRoot == "" {
			pctx.recentRoot = pctx.headRoot
		}
	}

	for _, key := range missing {
		val := resolvePlaceholder(key, pctx)
		if val != "" {
			params[key] = val
		}
	}

	finalPath = pathTpl
	for k, v := range params {
		finalPath = strings.ReplaceAll(finalPath, "{"+k+"}", v)
	}

	return finalPath, params, err
}

// pathContext carries everything resolvePlaceholder needs to expand a
// `{placeholder}` token. It is built once per task execution by
// querying the first ready client; every placeholder lookup then
// reads from it without further RPC traffic.
type pathContext struct {
	headSlot, headEpoch uint64
	headRoot            string
	recentSlot          uint64
	recentEpoch         uint64
	recentRoot          string
	slotsPerEpoch       uint64
}

func resolvePlaceholder(key string, pctx pathContext) string {
	// Handle {slot+N}, {epoch+N}, {recent_slot-N}, etc. The offset
	// suffix is parsed first so the rest of the function operates on
	// the bare keyword.
	base := key
	offset := int64(0)

	// `recent_*` keys contain an underscore, so look for +/- ONLY
	// after the keyword itself. A simple findFirst over the full key
	// would mis-parse `recent_slot-2` because `_` doesn't separate
	// the keyword from the offset.
	if idx := strings.IndexAny(key, "+-"); idx > 0 {
		base = key[:idx]

		var op rune
		if key[idx] == '+' {
			op = '+'
		} else {
			op = '-'
		}

		offsetStr := key[idx+1:]

		var n int64
		if _, err := fmt.Sscanf(offsetStr, "%d", &n); err == nil {
			if op == '+' {
				offset = n
			} else {
				offset = -n
			}
		}
	}

	switch base {
	case "slot":
		return fmt.Sprintf("%d", offsetUint64(pctx.headSlot, offset))
	case "epoch":
		return fmt.Sprintf("%d", offsetUint64(pctx.headEpoch, offset))
	case "recent_slot":
		return fmt.Sprintf("%d", offsetUint64(pctx.recentSlot, offset))
	case "recent_epoch":
		return fmt.Sprintf("%d", offsetUint64(pctx.recentEpoch, offset))
	case "block_id":
		return "head"
	case "state_id":
		return "head"
	case "beacon_block_root":
		return pctx.headRoot
	case "recent_block_root":
		return pctx.recentRoot
	case "builder_index":
		return "0"
	case "validator_index":
		return "0"
	}

	return ""
}

// offsetUint64 applies a signed offset to a uint64, clamping at zero on
// underflow so we never produce a negative slot/epoch number.
func offsetUint64(v uint64, offset int64) uint64 {
	if offset >= 0 {
		return v + uint64(offset)
	}

	neg := uint64(-offset)
	if neg > v {
		return 0
	}

	return v - neg
}

func buildMatrixRow(results []*PerClientResult) map[string]*MatrixCell {
	rank := map[string]int{
		resultSkipped: 0,
		resultPass:    1,
		resultPartial: 2,
		resultFail:    3,
	}
	out := map[string]*MatrixCell{}

	for _, r := range results {
		if r == nil {
			continue
		}

		key := r.ClientType
		if key == "" {
			key = clientTypeUnk
		}

		existing, ok := out[key]
		if !ok || rank[r.Status] > rank[existing.Result] {
			out[key] = &MatrixCell{
				Result:     r.Status,
				Note:       r.Note,
				HTTPStatus: r.HTTPStatus,
			}
		}
	}

	return out
}

// checkClient executes the configured probe against a single client and
// returns its classified result.
func (t *Task) checkClient(
	ctx context.Context,
	client *clients.PoolClient,
	resolvedPath string,
	resolvedParams map[string]string,
	successSchema, errorSchema, eventSchema *jsonschema.Schema,
) *PerClientResult {
	_ = resolvedParams // kept for future per-client placeholder overrides

	r := &PerClientResult{
		Client:     client.Config.Name,
		ClientType: clientTypeString(client),
	}

	if client.ConsensusClient == nil {
		r.Status = resultFail
		r.Error = "client has no consensus endpoint"

		return r
	}

	// Skip clients that don't have the required fork active.
	if t.config.RequireForkActive != "" {
		if !isForkActive(client.ConsensusClient, t.config.RequireForkActive) {
			r.Status = resultSkipped
			r.Note = fmt.Sprintf("fork %q not active", t.config.RequireForkActive)

			return r
		}
	}

	endpointConfig := client.ConsensusClient.GetEndpointConfig()
	if endpointConfig == nil || endpointConfig.URL == "" {
		r.Status = resultFail
		r.Error = "client has no URL configured"

		return r
	}

	if t.config.SSE != nil {
		return t.checkClientSSE(ctx, client, endpointConfig.URL, eventSchema)
	}

	return t.checkClientHTTP(ctx, client, endpointConfig.URL, resolvedPath, successSchema, errorSchema)
}

func (t *Task) checkClientHTTP(
	ctx context.Context,
	client *clients.PoolClient,
	baseURL, resolvedPath string,
	successSchema, errorSchema *jsonschema.Schema,
) *PerClientResult {
	r := &PerClientResult{
		Client:     client.Config.Name,
		ClientType: clientTypeString(client),
	}

	method := strings.ToUpper(t.config.Method)
	if method == "" {
		method = methodGet
	}

	// Build URL.
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + resolvedPath)
	if err != nil {
		r.Status = resultFail
		r.Error = fmt.Sprintf("invalid URL: %v", err)

		return r
	}

	if len(t.config.QueryParams) > 0 {
		q := u.Query()
		for k, v := range t.config.QueryParams {
			q.Set(k, v)
		}

		u.RawQuery = q.Encode()
	}

	r.URL = u.String()

	// Build body.
	var (
		body      io.Reader
		bodyBytes []byte
	)

	if t.config.BodyRaw != "" {
		bodyBytes = []byte(t.config.BodyRaw)
	} else if t.config.Body != nil {
		bodyBytes, err = json.Marshal(t.config.Body)
		if err != nil {
			r.Status = resultFail
			r.Error = fmt.Sprintf("invalid body: %v", err)

			return r
		}
	}

	if bodyBytes != nil {
		body = bytes.NewReader(bodyBytes)
	}

	reqCtx, cancel := context.WithTimeout(ctx, t.cfgRequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, method, u.String(), body)
	if err != nil {
		r.Status = resultFail
		r.Error = fmt.Sprintf("could not build request: %v", err)

		return r
	}

	// Default Content-Type when body sent.
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Per-client base headers from ClientConfig (e.g. auth).
	if cfgHeaders := client.ConsensusClient.GetEndpointConfig().Headers; len(cfgHeaders) > 0 {
		for k, v := range cfgHeaders {
			req.Header.Set(k, v)
		}
	}

	// User-supplied headers (override).
	for k, v := range t.config.Headers {
		req.Header.Set(k, v)
	}

	t0 := time.Now()
	httpClient := &http.Client{Timeout: t.cfgRequestTimeout()}

	resp, err := httpClient.Do(req)
	r.DurationMs = time.Since(t0).Milliseconds()

	if err != nil {
		r.Status = resultFail
		r.Error = fmt.Sprintf("request error: %v", err)

		return r
	}

	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseLen))
	r.HTTPStatus = resp.StatusCode

	classifyHTTPResult(r, &t.config, successSchema, errorSchema, respBytes)

	return r
}

func classifyHTTPResult(
	r *PerClientResult,
	cfg *Config,
	successSchema, errorSchema *jsonschema.Schema,
	body []byte,
) {
	if !containsInt(cfg.ExpectStatuses, r.HTTPStatus) {
		r.Status = resultFail
		r.Note = fmt.Sprintf("unexpected status %d", r.HTTPStatus)
		// Try to capture short error message for debugging.
		if len(body) > 0 {
			snippet := strings.TrimSpace(string(body))
			if len(snippet) > 240 {
				snippet = snippet[:240] + "…"
			}

			r.Error = snippet
		}

		return
	}

	if cfg.IgnoreSchema {
		r.Status = resultPass
		r.Note = fmt.Sprintf("status %d (schema ignored)", r.HTTPStatus)

		return
	}

	if containsInt(cfg.SuccessStatuses, r.HTTPStatus) {
		if successSchema == nil {
			r.Status = resultPass
			r.Note = fmt.Sprintf("status %d (no schema)", r.HTTPStatus)

			return
		}

		errs := validateBytes(successSchema, body)
		if len(errs) == 0 {
			r.Status = resultPass

			return
		}

		r.Status = resultPartial
		r.SchemaErrors = errs
		r.Note = "response body does not match success schema"

		return
	}

	if containsInt(cfg.ErrorStatuses, r.HTTPStatus) {
		if errorSchema == nil {
			r.Status = resultPass
			r.Note = fmt.Sprintf("status %d (no error schema)", r.HTTPStatus)

			return
		}

		errs := validateBytes(errorSchema, body)
		if len(errs) == 0 {
			r.Status = resultPass
			r.Note = fmt.Sprintf("well-formed error status %d", r.HTTPStatus)

			return
		}

		r.Status = resultPartial
		r.SchemaErrors = errs
		r.Note = fmt.Sprintf("error status %d but body does not match ErrorMessage schema", r.HTTPStatus)

		return
	}

	// Fell through expect list but not in success/error → treat as partial.
	r.Status = resultPartial
	r.Note = fmt.Sprintf("status %d not in success/error sets", r.HTTPStatus)
}

func clientTypeString(client *clients.PoolClient) string {
	if client == nil || client.ConsensusClient == nil {
		return clientTypeUnk
	}

	return client.ConsensusClient.GetClientType().String()
}

func isForkActive(c *consensus.Client, forkName string) bool {
	// Best effort: check that the client's head has reached or passed the fork
	// epoch. We use the consensus pool's chain spec / fork schedule.
	headSlot, _ := c.GetLastHead()
	if headSlot == 0 {
		return false
	}

	// Without a deep coupling to the pool's chain spec we approximate by
	// asking the RPC client for the head's beacon-block version header.
	// If we can't determine it, return true so we don't skip incorrectly —
	// most playbooks will gate this on chain state separately.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	specs, err := c.GetRPCClient().GetConfigSpecs(ctx)
	if err != nil {
		return true
	}

	forkEpochKey := strings.ToUpper(forkName) + "_FORK_EPOCH"

	forkEpochRaw, ok := specs[forkEpochKey]
	if !ok {
		return true
	}

	var forkEpoch uint64

	switch v := forkEpochRaw.(type) {
	case uint64:
		forkEpoch = v
	case int64:
		if v < 0 {
			return true
		}

		forkEpoch = uint64(v)
	case float64:
		if v < 0 {
			return true
		}

		forkEpoch = uint64(v)
	default:
		return true
	}

	slotsPerEpoch := uint64(32)

	if v, ok := specs["SLOTS_PER_EPOCH"]; ok {
		if vu, ok := v.(uint64); ok && vu > 0 {
			slotsPerEpoch = vu
		}
	}

	return uint64(headSlot) >= forkEpoch*slotsPerEpoch
}

func containsInt(xs []int, n int) bool {
	for _, x := range xs {
		if x == n {
			return true
		}
	}

	return false
}
