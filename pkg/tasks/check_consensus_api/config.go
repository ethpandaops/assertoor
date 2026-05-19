package checkconsensusapi

import (
	"fmt"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

// Config holds all options for the check_consensus_api task. The task hits a
// single beacon-API endpoint (HTTP) or subscribes to a single Server-Sent-Event
// topic against every connected CL client (or a filtered subset), classifies
// each per-client response, and emits both per-client results and an
// aggregated matrix row.
type Config struct {
	// Identification & display ------------------------------------------------
	RowID        string `yaml:"rowId" json:"rowId" desc:"Stable short identifier for this check. Used by the aggregator/matrix tasks."`
	RowTitle     string `yaml:"rowTitle" json:"rowTitle" desc:"Human-friendly title for this check (used in matrix tables)."`
	ReferenceURL string `yaml:"referenceUrl" json:"referenceUrl" desc:"Reference URL (e.g. spec PR) for this check."`

	// Client selection --------------------------------------------------------
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific CL endpoints (matches client.Name). Empty = all."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude specific CL endpoints (matches client.Name)."`

	// HTTP request (omit for SSE) --------------------------------------------
	Method      string            `yaml:"method" json:"method" desc:"HTTP method. Default GET. Set to empty when 'sse' is configured."`
	Path        string            `yaml:"path" json:"path" desc:"Endpoint path, e.g. '/eth/v2/beacon/blocks'. Supports placeholders like {slot}, {epoch}, {block_id}, {builder_index}, {beacon_block_root}."`
	PathParams  map[string]string `yaml:"pathParams" json:"pathParams" desc:"Explicit path placeholder overrides. Values are templated into 'path'."`
	QueryParams map[string]string `yaml:"queryParams" json:"queryParams" desc:"Query string parameters."`
	Headers     map[string]string `yaml:"headers" json:"headers" desc:"Request headers."`
	Body        interface{}       `yaml:"body" json:"body" desc:"Request body. Sent as application/json unless Content-Type header is set."`
	BodyRaw     string            `yaml:"bodyRaw" json:"bodyRaw" desc:"Raw request body (alternative to 'body'). Useful for SSZ bytes or non-JSON payloads."`

	// SSE (omit for HTTP) -----------------------------------------------------
	SSE *SSEConfig `yaml:"sse" json:"sse" desc:"If set, run this check as an SSE subscription instead of an HTTP request."`

	// Classification ----------------------------------------------------------
	ExpectStatuses  []int `yaml:"expectStatuses" json:"expectStatuses" desc:"HTTP statuses indicating the endpoint exists (responder recognized the route). Default: [200, 400, 404, 415, 503]."`
	SuccessStatuses []int `yaml:"successStatuses" json:"successStatuses" desc:"HTTP statuses where 'responseSchema' is applied. Default: [200]."`
	ErrorStatuses   []int `yaml:"errorStatuses" json:"errorStatuses" desc:"HTTP statuses where 'errorSchema' is applied. Default: [400, 404, 415, 503]."`

	// Validation schemas (inline JSON Schema) ---------------------------------
	ResponseSchema map[string]interface{} `yaml:"responseSchema" json:"responseSchema" desc:"Inline JSON Schema for the success response body."`
	ErrorSchema    map[string]interface{} `yaml:"errorSchema" json:"errorSchema" desc:"Inline JSON Schema for documented error responses (typically the ErrorMessage shape)."`
	EventSchema    map[string]interface{} `yaml:"eventSchema" json:"eventSchema" desc:"Inline JSON Schema for the SSE event data payload (after 'data: ' prefix)."`

	// Behavior ----------------------------------------------------------------
	RequestTimeout    helper.Duration `yaml:"requestTimeout" json:"requestTimeout" desc:"Per-client HTTP/SSE request timeout. Default 30s."`
	OverallTimeout    helper.Duration `yaml:"overallTimeout" json:"overallTimeout" desc:"Maximum overall wallclock duration. Default 90s."`
	FailOnAllError    bool            `yaml:"failOnAllError" json:"failOnAllError" desc:"If true, set task result = failure when zero clients pass."`
	FailOnAnyError    bool            `yaml:"failOnAnyError" json:"failOnAnyError" desc:"If true, set task result = failure when any client fails/partial."`
	RequireForkActive string          `yaml:"requireForkActive" json:"requireForkActive" desc:"If set (e.g. 'gloas'), each client where this fork is not active records 'skipped'."`
	IgnoreSchema      bool            `yaml:"ignoreSchema" json:"ignoreSchema" desc:"If true, skip schema validation (any status in expectStatuses passes)."`
	Concurrency       int             `yaml:"concurrency" json:"concurrency" desc:"Maximum number of clients hit in parallel. Default 6."`
}

type SSEConfig struct {
	Topic          string          `yaml:"topic" json:"topic" desc:"SSE topic name to subscribe to (single topic per check)."`
	TimeoutSeconds int             `yaml:"timeoutSeconds" json:"timeoutSeconds" desc:"How long to wait for events after subscribing. Default 36 (≈3 slots)."`
	MinEvents      int             `yaml:"minEvents" json:"minEvents" desc:"Minimum number of matching events to receive for a 'pass'. Default 1."`
	EventName      string          `yaml:"eventName" json:"eventName" desc:"Override SSE 'event:' name to match (defaults to 'topic')."`
	SubscribeWait  helper.Duration `yaml:"subscribeWait" json:"subscribeWait" desc:"Extra wait after subscription open before counting events. Default 0."`
}

func DefaultConfig() Config {
	return Config{
		Method:          "GET",
		ExpectStatuses:  []int{200, 400, 404, 415, 503},
		SuccessStatuses: []int{200},
		ErrorStatuses:   []int{400, 404, 415, 503},
		RequestTimeout:  helper.Duration{Duration: defaultRequestTimeout},
		OverallTimeout:  helper.Duration{Duration: defaultOverallTimeout},
		Concurrency:     6,
	}
}

func (c *Config) Validate() error {
	if c.RowID == "" {
		return fmt.Errorf("rowId is required")
	}

	if c.SSE != nil {
		if c.SSE.Topic == "" {
			return fmt.Errorf("sse.topic is required when sse is configured")
		}

		if c.SSE.TimeoutSeconds <= 0 {
			c.SSE.TimeoutSeconds = defaultSSETimeoutSeconds
		}

		if c.SSE.MinEvents <= 0 {
			c.SSE.MinEvents = 1
		}

		return nil
	}

	if c.Path == "" {
		return fmt.Errorf("path is required for HTTP checks")
	}

	if c.Method == "" {
		c.Method = "GET"
	}

	return nil
}
