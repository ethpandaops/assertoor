package checkhttpjson

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

const (
	testJSONURL        = "http://localhost:8080/api"
	testAssertionName  = "test"
	testAssertionName2 = "test2"
	testInvalidOpLabel = "invalid operator"
	testMethodGET      = "GET"
	testMethodPOST     = "POST"
	testQueryReady     = ".ready"
	testQueryStatus    = ".status"
	testQueryValue     = ".value"
	testQueryItems     = ".items"
	testHello          = "hello"
	testWorld          = "world"
	testHelloWorld     = "hello world"
	testValueKey       = "value"
	testReadyCheck     = "ready_check"
	testStatusCheck    = "status_check"
	testItemsExist     = "items_exist"
	testCheck          = "check"
	testStatusOK       = "ok"
	testCount          = "count"
	testService        = "service"
	testLatencyMS      = "latency_ms"
	testStatus         = "status"
	testID             = "id"
	testName           = "name"
	testItems          = "items"
	testQueryItemsAll  = ".items[]"
)

// validBaseConfig returns a config with required fields for validation to pass.
func validBaseConfig() Config {
	return Config{
		Method:         testMethodGET,
		PollInterval:   helper.Duration{Duration: 10 * time.Second},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
	}
}

func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// Config Validation Tests
// =============================================================================

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name       string
		configFunc func() Config
		wantErr    string
	}{
		{
			name:       "missing url",
			configFunc: func() Config { return Config{} },
			wantErr:    "url is required",
		},
		{
			name: "empty assertions allowed for status-only check",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{}

				return c
			},
			wantErr: "",
		},
		{
			name: "missing assertion name",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Query: testQueryReady, Operator: OperatorEq, Value: true},
				}

				return c
			},
			wantErr: "assertion[0]: name is required",
		},
		{
			name: "duplicate assertion names",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryReady, Operator: OperatorEq, Value: true},
					{Name: testAssertionName, Query: testQueryStatus, Operator: OperatorEq, Value: testStatusOK},
				}

				return c
			},
			wantErr: "assertion[1]: duplicate name",
		},
		{
			name: "missing query",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Operator: OperatorEq, Value: true},
				}

				return c
			},
			wantErr: "query is required",
		},
		{
			name: testInvalidOpLabel,
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryValue, Operator: "invalid", Value: 0},
				}

				return c
			},
			wantErr: "invalid operator",
		},
		{
			name: "invalid jq syntax",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: ".value[", Operator: OperatorEq, Value: 0},
				}

				return c
			},
			wantErr: "invalid jq syntax",
		},
		{
			name: "both exists and operator set",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryValue, Exists: boolPtr(true), Operator: OperatorEq, Value: 0},
				}

				return c
			},
			wantErr: "cannot set both 'exists' and 'operator'",
		},
		{
			name: "neither exists nor operator set",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryValue},
				}

				return c
			},
			wantErr: "must set either 'exists' or 'operator'",
		},
		{
			name: "both expectStatus and expectStatuses set",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				status := 200
				c.ExpectStatus = &status
				c.ExpectStatuses = []int{200, 201}

				return c
			},
			wantErr: "cannot set both expectStatus and expectStatuses",
		},
		{
			name: "HEAD with assertions",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Method = MethodHead
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryValue, Operator: OperatorEq, Value: 0},
				}

				return c
			},
			wantErr: "HEAD requests cannot have assertions",
		},
		{
			name: "HEAD without assertions is valid",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Method = MethodHead
				c.Assertions = []AssertionConfig{}

				return c
			},
			wantErr: "",
		},
		{
			name: "invalid method",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Method = "INVALID"

				return c
			},
			wantErr: "invalid method",
		},
		{
			name: "invalid maxResponseSize",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.MaxResponseSize = "invalid"

				return c
			},
			wantErr: "invalid maxResponseSize",
		},
		{
			name: "valid config with multiple assertions",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryReady, Operator: OperatorEq, Value: true},
					{Name: testAssertionName2, Query: testQueryItems, Exists: boolPtr(true)},
				}

				return c
			},
			wantErr: "",
		},
		{
			name: "operator without value",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Query: testQueryValue, Operator: OperatorEq},
				}

				return c
			},
			wantErr: "'value' is required when 'operator' is set",
		},
		{
			name: "invalid expectStatus below 100",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				status := 99
				c.ExpectStatus = &status

				return c
			},
			wantErr: "invalid expectStatus 99: must be between 100 and 599",
		},
		{
			name: "invalid expectStatus above 599",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				status := 600
				c.ExpectStatus = &status

				return c
			},
			wantErr: "invalid expectStatus 600: must be between 100 and 599",
		},
		{
			name: "invalid expectStatuses contains zero",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.ExpectStatuses = []int{200, 0, 201}

				return c
			},
			wantErr: "invalid expectStatuses value 0: must be between 100 and 599",
		},
		{
			name: "valid expectStatuses",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testJSONURL
				c.ExpectStatuses = []int{200, 201, 404, 500}

				return c
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()
			err := cfg.Validate()

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}

// =============================================================================
// Operator Tests
// =============================================================================

func TestEvaluateOperator(t *testing.T) {
	tests := []struct {
		name     string
		op       Operator
		actual   any
		expected any
		want     bool
		wantErr  bool
	}{
		// eq/neq tests
		{name: "eq bool true", op: OperatorEq, actual: true, expected: true, want: true},
		{name: "eq bool false", op: OperatorEq, actual: true, expected: false, want: false},
		{name: "eq string", op: OperatorEq, actual: testHello, expected: testHello, want: true},
		{name: "eq string mismatch", op: OperatorEq, actual: testHello, expected: testWorld, want: false},
		{name: "eq number int", op: OperatorEq, actual: float64(42), expected: float64(42), want: true},
		{name: "neq bool", op: OperatorNeq, actual: true, expected: false, want: true},
		{name: "neq string", op: OperatorNeq, actual: testHello, expected: testWorld, want: true},

		// numeric type coercion tests (JSON returns float64, YAML may return int)
		{name: "eq float64 vs int", op: OperatorEq, actual: float64(42), expected: 42, want: true},
		{name: "eq int vs float64", op: OperatorEq, actual: 42, expected: float64(42), want: true},
		{name: "eq float64 vs int64", op: OperatorEq, actual: float64(100), expected: int64(100), want: true},
		{name: "neq float64 vs int different", op: OperatorNeq, actual: float64(42), expected: 43, want: true},
		{name: "neq float64 vs int same", op: OperatorNeq, actual: float64(42), expected: 42, want: false},

		// numeric comparison tests
		{name: "gt true", op: OperatorGt, actual: float64(10), expected: float64(5), want: true},
		{name: "gt false", op: OperatorGt, actual: float64(5), expected: float64(10), want: false},
		{name: "gt equal", op: OperatorGt, actual: float64(5), expected: float64(5), want: false},
		{name: "gte true greater", op: OperatorGte, actual: float64(10), expected: float64(5), want: true},
		{name: "gte true equal", op: OperatorGte, actual: float64(5), expected: float64(5), want: true},
		{name: "gte false", op: OperatorGte, actual: float64(4), expected: float64(5), want: false},
		{name: "lt true", op: OperatorLt, actual: float64(5), expected: float64(10), want: true},
		{name: "lt false", op: OperatorLt, actual: float64(10), expected: float64(5), want: false},
		{name: "lte true less", op: OperatorLte, actual: float64(5), expected: float64(10), want: true},
		{name: "lte true equal", op: OperatorLte, actual: float64(5), expected: float64(5), want: true},
		{name: "lte false", op: OperatorLte, actual: float64(10), expected: float64(5), want: false},

		// numeric type coercion
		{name: "gt int types", op: OperatorGt, actual: 10, expected: 5, want: true},
		{name: "gt int vs float", op: OperatorGt, actual: float64(10), expected: 5, want: true},

		// type mismatch for numeric ops
		{name: "gt string error", op: OperatorGt, actual: testHello, expected: float64(5), wantErr: true},
		{name: "gt expected string error", op: OperatorGt, actual: float64(5), expected: testHello, wantErr: true},

		// contains tests
		{name: "contains string", op: OperatorContains, actual: testHelloWorld, expected: testWorld, want: true},
		{name: "contains string miss", op: OperatorContains, actual: testHelloWorld, expected: "foo", want: false},
		{name: "contains array", op: OperatorContains, actual: []any{"a", "b", "c"}, expected: "b", want: true},
		{name: "contains array miss", op: OperatorContains, actual: []any{"a", "b", "c"}, expected: "d", want: false},
		{name: "contains object", op: OperatorContains, actual: map[string]any{"a": 1, "b": 2}, expected: map[string]any{"a": 1}, want: true},
		{name: "contains object miss", op: OperatorContains, actual: map[string]any{"a": 1, "b": 2}, expected: map[string]any{"c": 3}, want: false},

		// contains with numeric type coercion
		{name: "contains array float64 vs int", op: OperatorContains, actual: []any{float64(1), float64(2), float64(3)}, expected: 2, want: true},
		{name: "contains object float64 vs int", op: OperatorContains, actual: map[string]any{testCount: float64(42)}, expected: map[string]any{testCount: 42}, want: true},

		// recursive numeric coercion in nested structures
		{
			name: "eq nested map float64 vs int",
			op:   OperatorEq,
			actual: map[string]any{
				testService: map[string]any{
					testLatencyMS: float64(5),
					testStatus:    testStatusOK,
				},
			},
			expected: map[string]any{
				testService: map[string]any{
					testLatencyMS: 5,
					testStatus:    testStatusOK,
				},
			},
			want: true,
		},
		{
			name: "eq deeply nested numeric",
			op:   OperatorEq,
			actual: map[string]any{
				"outer": map[string]any{
					"inner": map[string]any{
						testCount: float64(42),
					},
				},
			},
			expected: map[string]any{
				"outer": map[string]any{
					"inner": map[string]any{
						testCount: 42,
					},
				},
			},
			want: true,
		},
		{
			name:     "eq nested array float64 vs int",
			op:       OperatorEq,
			actual:   []any{float64(1), float64(2), []any{float64(3), float64(4)}},
			expected: []any{1, 2, []any{3, 4}},
			want:     true,
		},
		{
			name: "eq array of objects with numeric coercion",
			op:   OperatorEq,
			actual: []any{
				map[string]any{testID: float64(1), testName: "a"},
				map[string]any{testID: float64(2), testName: "b"},
			},
			expected: []any{
				map[string]any{testID: 1, testName: "a"},
				map[string]any{testID: 2, testName: "b"},
			},
			want: true,
		},
		{
			name: "neq nested map different values",
			op:   OperatorNeq,
			actual: map[string]any{
				testService: map[string]any{testLatencyMS: float64(5)},
			},
			expected: map[string]any{
				testService: map[string]any{testLatencyMS: 10},
			},
			want: true,
		},

		// not_contains tests
		{name: "not_contains string", op: OperatorNotContains, actual: testHelloWorld, expected: "foo", want: true},
		{name: "not_contains string miss", op: OperatorNotContains, actual: testHelloWorld, expected: testWorld, want: false},

		// contains type mismatch
		{name: "contains string with int", op: OperatorContains, actual: testHello, expected: 123, wantErr: true},
		{name: "contains int error", op: OperatorContains, actual: 123, expected: 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateOperator(tt.op, tt.actual, tt.expected)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// HTTP Server Tests
// =============================================================================

func newTestTaskWithContext(cfg *Config) (*Task, *types.TaskResult) {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	outputs := vars.NewVariables(nil)

	var lastResult types.TaskResult

	ctx := &types.TaskContext{
		Outputs:        outputs,
		SetResult:      func(r types.TaskResult) { lastResult = r },
		ReportProgress: func(_ float64, _ string) {},
	}

	var config Config
	if cfg != nil {
		config = *cfg
	}

	// Ensure maxResponseSizeBytes has a default for tests that don't call Validate
	// Use decimal value (10MB = 10,000,000) consistent with humanize.ParseBytes
	if config.maxResponseSizeBytes == 0 {
		config.maxResponseSizeBytes = 10 * 1000 * 1000
	}

	task := &Task{
		ctx:    ctx,
		config: config,
		logger: logger,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout.Duration,
		},
	}

	return task, &lastResult
}

func TestTask_StatusOnlyCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         testMethodGET,
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		Assertions:     []AssertionConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("expected success, got %v", *result)
	}
}

func TestTask_HEADRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != MethodHead {
			t.Errorf("expected HEAD request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         MethodHead,
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		Assertions:     []AssertionConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("expected success, got %v", *result)
	}
}

func TestTask_UnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := Config{
		URL:             server.URL,
		Method:          testMethodGET,
		PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		FailOnCheckMiss: true,
		Assertions:      []AssertionConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err == nil {
		t.Error("expected error for unexpected status")
	}

	if *result != types.TaskResultFailure {
		t.Errorf("expected failure, got %v", *result)
	}
}

func TestTask_MultipleExpectedStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         testMethodGET,
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		ExpectStatuses: []int{200, 201, 202},
		Assertions:     []AssertionConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("expected success for status 201, got %v", *result)
	}
}

func TestTask_JSONAssertions(t *testing.T) {
	jsonResponse := map[string]any{
		"ready":   true,
		"status":  testStatusOK,
		"count":   float64(42),
		testItems: []any{"a", "b", "c"},
		"nested": map[string]any{
			testValueKey: float64(100),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jsonResponse)
	}))
	defer server.Close()

	tests := []struct {
		name       string
		assertions []AssertionConfig
		wantPass   bool
	}{
		{
			name: "eq bool passes",
			assertions: []AssertionConfig{
				{Name: testReadyCheck, Query: testQueryReady, Operator: OperatorEq, Value: true},
			},
			wantPass: true,
		},
		{
			name: "eq string passes",
			assertions: []AssertionConfig{
				{Name: testStatusCheck, Query: testQueryStatus, Operator: OperatorEq, Value: testStatusOK},
			},
			wantPass: true,
		},
		{
			name: "gt number passes",
			assertions: []AssertionConfig{
				{Name: "count_check", Query: ".count", Operator: OperatorGt, Value: float64(40)},
			},
			wantPass: true,
		},
		{
			name: "exists true passes",
			assertions: []AssertionConfig{
				{Name: testItemsExist, Query: testQueryItems, Exists: boolPtr(true)},
			},
			wantPass: true,
		},
		{
			name: "exists false passes for missing",
			assertions: []AssertionConfig{
				{Name: "missing_check", Query: ".nonexistent", Exists: boolPtr(false)},
			},
			wantPass: true,
		},
		{
			name: "nested query passes",
			assertions: []AssertionConfig{
				{Name: "nested_check", Query: ".nested.value", Operator: OperatorEq, Value: float64(100)},
			},
			wantPass: true,
		},
		{
			name: "array length check",
			assertions: []AssertionConfig{
				// gojq returns int for length, so we use int for comparison
				{Name: "items_length", Query: ".items | length", Operator: OperatorEq, Value: 3},
			},
			wantPass: true,
		},
		{
			name: "eq fails on mismatch",
			assertions: []AssertionConfig{
				{Name: "wrong_status", Query: testQueryStatus, Operator: OperatorEq, Value: "error"},
			},
			wantPass: false,
		},
		{
			name: "exists true fails for missing",
			assertions: []AssertionConfig{
				{Name: "missing_required", Query: ".nonexistent", Exists: boolPtr(true)},
			},
			wantPass: false,
		},
		{
			name: "multiple assertions all pass",
			assertions: []AssertionConfig{
				{Name: testReadyCheck, Query: testQueryReady, Operator: OperatorEq, Value: true},
				{Name: testStatusCheck, Query: testQueryStatus, Operator: OperatorEq, Value: testStatusOK},
				{Name: testItemsExist, Query: testQueryItems, Exists: boolPtr(true)},
			},
			wantPass: true,
		},
		{
			name: "multiple assertions one fails",
			assertions: []AssertionConfig{
				{Name: testReadyCheck, Query: testQueryReady, Operator: OperatorEq, Value: true},
				{Name: "wrong_check", Query: testQueryStatus, Operator: OperatorEq, Value: "error"},
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				URL:             server.URL,
				Method:          testMethodGET,
				PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
				RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
				FailOnCheckMiss: true,
				Assertions:      tt.assertions,
			}
			if err := cfg.Validate(); err != nil {
				t.Fatalf("config validation failed: %v", err)
			}

			task, result := newTestTaskWithContext(&cfg)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := task.Execute(ctx)

			if tt.wantPass {
				if err != nil {
					t.Errorf("expected pass, got error: %v", err)
				}

				if *result != types.TaskResultSuccess {
					t.Errorf("expected success, got %v", *result)
				}
			} else {
				if err == nil {
					t.Error("expected error for failed assertion")
				}

				if *result != types.TaskResultFailure {
					t.Errorf("expected failure, got %v", *result)
				}
			}
		})
	}
}

func TestTask_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	cfg := Config{
		URL:             server.URL,
		Method:          testMethodGET,
		PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		FailOnCheckMiss: true,
		Assertions: []AssertionConfig{
			{Name: testCheck, Query: testQueryValue, Operator: OperatorEq, Value: true},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "JSON parse error") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}

	if task.parseErrors != 1 {
		t.Errorf("expected parseErrors=1, got %d", task.parseErrors)
	}
}

func TestTask_EmptyBodyWithAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty body
	}))
	defer server.Close()

	cfg := Config{
		URL:             server.URL,
		Method:          testMethodGET,
		PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		FailOnCheckMiss: true,
		Assertions: []AssertionConfig{
			{Name: testCheck, Query: testQueryValue, Operator: OperatorEq, Value: true},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err == nil {
		t.Error("expected error for empty body with assertions")
	}

	if task.parseErrors != 1 {
		t.Errorf("expected parseErrors=1, got %d", task.parseErrors)
	}
}

func TestTask_RequestBody(t *testing.T) {
	var receivedBody map[string]any

	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")

		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         testMethodPOST,
		Body:           map[string]any{"key": testValueKey},
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		Assertions: []AssertionConfig{
			{Name: "success", Query: ".success", Operator: OperatorEq, Value: true},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedContentType)
	}

	if receivedBody["key"] != testValueKey {
		t.Errorf("expected body key=value, got %v", receivedBody)
	}
}

func TestTask_BodyRawTakesPrecedence(t *testing.T) {
	var receivedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1024)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         testMethodPOST,
		Body:           map[string]any{"ignored": true},
		BodyRaw:        "raw body content",
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		Assertions:     []AssertionConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedBody != "raw body content" {
		t.Errorf("expected raw body content, got %s", receivedBody)
	}
}

func TestTask_CustomHeaders(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         testMethodGET,
		Headers:        map[string]string{"Authorization": "Bearer token123"},
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		Assertions:     []AssertionConfig{},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer token123" {
		t.Errorf("expected Authorization header, got %s", receivedAuth)
	}
}

func TestTask_MaxResponseSize(t *testing.T) {
	largeBody := strings.Repeat("x", 1024*1024) // 1MB

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	cfg := Config{
		URL:             server.URL,
		Method:          testMethodGET,
		MaxResponseSize: "1KB",
		PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		FailOnCheckMiss: true,
		Assertions: []AssertionConfig{
			{Name: testCheck, Query: ".", Exists: boolPtr(true)},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err == nil {
		t.Error("expected error for oversized response")
	}

	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Errorf("expected max size error, got: %v", err)
	}
}

func TestTask_AllowMissingOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"existing": true}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:             server.URL,
		Method:          testMethodGET,
		PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		FailOnCheckMiss: true,
		Assertions: []AssertionConfig{
			{Name: "missing_allowed", Query: ".missing", Operator: OperatorEq, Value: true, AllowMissing: boolPtr(true)},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected pass with allowMissing, got error: %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("expected success with allowMissing, got %v", *result)
	}
}

func TestTask_MultipleQueryResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items": [1, 2, 3]}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:             server.URL,
		Method:          testMethodGET,
		PollInterval:    helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		FailOnCheckMiss: true,
		Assertions: []AssertionConfig{
			// This query returns multiple results
			{Name: "items_check", Query: testQueryItemsAll, Operator: OperatorEq, Value: float64(1)},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err == nil {
		t.Error("expected error for multiple query results")
	}

	// The error is wrapped at top level as "assertions failed"
	if !strings.Contains(err.Error(), "assertions failed") {
		t.Errorf("expected assertions failed error, got: %v", err)
	}
}

func TestTask_ExistsWithMultipleResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items": [1, 2, 3]}`))
	}))
	defer server.Close()

	cfg := Config{
		URL:            server.URL,
		Method:         testMethodGET,
		PollInterval:   helper.Duration{Duration: 100 * time.Millisecond},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
		Assertions: []AssertionConfig{
			// exists mode allows multiple results
			{Name: testItemsExist, Query: testQueryItemsAll, Exists: boolPtr(true)},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("config validation failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected pass for exists with multiple results, got error: %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("expected success, got %v", *result)
	}
}

// =============================================================================
// Full-Cycle Integration Tests
// =============================================================================
// These tests verify the complete flow from DefaultConfig() through Validate()
// to execution, mirroring how the task is used in production.

func TestIntegration_BasicAssertion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready": true, "version": "1.0.0", "count": 42}`))
	}))
	defer server.Close()

	// Start from DefaultConfig (like production)
	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Assertions = []AssertionConfig{
		{
			Name:     "service_ready",
			Query:    ".ready",
			Operator: OperatorEq,
			Value:    true,
		},
	}

	// Validate (like production)
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Verify maxResponseSizeBytes was set by Validate
	if cfg.GetMaxResponseSizeBytes() == 0 {
		t.Fatal("maxResponseSizeBytes should be set after Validate()")
	}

	// Create task with context (like production)
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	outputs := vars.NewVariables(nil)

	var lastResult types.TaskResult

	ctx := &types.TaskContext{
		Outputs:        outputs,
		SetResult:      func(r types.TaskResult) { lastResult = r },
		ReportProgress: func(_ float64, _ string) {},
	}

	task := &Task{
		ctx:    ctx,
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout.Duration,
		},
	}

	// Execute
	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(bgCtx)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if lastResult != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", lastResult)
	}

	// Verify outputs
	passed := outputs.GetVar("passedAssertions")
	if passed == nil {
		t.Fatal("passedAssertions output not set")
	}

	passedList, ok := passed.([]any)
	if !ok {
		t.Fatalf("passedAssertions type = %T, want []any", passed)
	}

	if len(passedList) != 1 || passedList[0] != "service_ready" {
		t.Errorf("passedAssertions = %v, want [service_ready]", passedList)
	}
}

func TestIntegration_ExistsMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"proof_types": [
				{"proof_type": "reth-zisk", "can_verify": true},
				{"proof_type": "other", "can_verify": false}
			]
		}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Assertions = []AssertionConfig{
		{
			Name:   "reth_zisk_verifier_loaded",
			Query:  `.proof_types[] | select(.proof_type == "reth-zisk" and .can_verify == true)`,
			Exists: boolPtr(true),
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	outputs := vars.NewVariables(nil)

	var lastResult types.TaskResult

	ctx := &types.TaskContext{
		Outputs:        outputs,
		SetResult:      func(r types.TaskResult) { lastResult = r },
		ReportProgress: func(_ float64, _ string) {},
	}

	task := &Task{
		ctx:    ctx,
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout.Duration,
		},
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(bgCtx)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if lastResult != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", lastResult)
	}
}

func TestIntegration_MultipleAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "healthy",
			"services": {
				"database": {"connected": true, "latency_ms": 5},
				"cache": {"connected": true, "latency_ms": 2}
			},
			"uptime_seconds": 86400
		}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Assertions = []AssertionConfig{
		{
			Name:     "status_healthy",
			Query:    ".status",
			Operator: OperatorEq,
			Value:    "healthy",
		},
		{
			Name:     "db_connected",
			Query:    ".services.database.connected",
			Operator: OperatorEq,
			Value:    true,
		},
		{
			Name:     "db_latency_ok",
			Query:    ".services.database.latency_ms",
			Operator: OperatorLt,
			Value:    float64(10),
		},
		{
			Name:     "uptime_sufficient",
			Query:    ".uptime_seconds",
			Operator: OperatorGte,
			Value:    float64(3600),
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	outputs := vars.NewVariables(nil)

	var lastResult types.TaskResult

	ctx := &types.TaskContext{
		Outputs:        outputs,
		SetResult:      func(r types.TaskResult) { lastResult = r },
		ReportProgress: func(_ float64, _ string) {},
	}

	task := &Task{
		ctx:    ctx,
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout.Duration,
		},
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(bgCtx)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if lastResult != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", lastResult)
	}

	// Verify all four passed
	passed := outputs.GetVar("passedAssertions")
	passedList, ok := passed.([]any)

	if !ok {
		t.Fatalf("passedAssertions type = %T, want []any", passed)
	}

	if len(passedList) != 4 {
		t.Errorf("passedAssertions count = %d, want 4", len(passedList))
	}

	// Verify values output
	values := outputs.GetVar("values")
	valuesMap, ok := values.(map[string]any)

	if !ok {
		t.Fatalf("values type = %T, want map[string]any", values)
	}

	if v := valuesMap["status_healthy"]; v != "healthy" {
		t.Errorf("values[status_healthy] = %v, want 'healthy'", v)
	}
}

func TestIntegration_StatusOnlyCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Return non-JSON response - status-only check should still pass
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Assertions = []AssertionConfig{} // Empty - status-only check

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	outputs := vars.NewVariables(nil)

	var lastResult types.TaskResult

	ctx := &types.TaskContext{
		Outputs:        outputs,
		SetResult:      func(r types.TaskResult) { lastResult = r },
		ReportProgress: func(_ float64, _ string) {},
	}

	task := &Task{
		ctx:    ctx,
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: cfg.RequestTimeout.Duration,
		},
	}

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(bgCtx)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if lastResult != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", lastResult)
	}

	// Verify httpStatus output
	httpStatus := outputs.GetVar("httpStatus")
	if httpStatus != 200 {
		t.Errorf("httpStatus = %v, want 200", httpStatus)
	}
}

// =============================================================================
// Max Results and Timeout Tests
// =============================================================================

func TestTask_MaxResultsExactly32Succeeds(t *testing.T) {
	// Generate JSON with exactly 32 items
	items := make([]int, 32)
	for i := range items {
		items[i] = i + 1
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{testItems: items})
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Assertions = []AssertionConfig{
		{
			Name:   "items_exist",
			Query:  ".items[]",
			Exists: boolPtr(true),
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected success with 32 results, got error: %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", *result)
	}
}

func TestTask_MaxResults33Fails(t *testing.T) {
	// Generate JSON with 33 items (exceeds max of 32)
	items := make([]int, 33)
	for i := range items {
		items[i] = i + 1
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{testItems: items})
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.FailOnCheckMiss = true
	cfg.Assertions = []AssertionConfig{
		{
			Name:   "items_exist",
			Query:  ".items[]",
			Exists: boolPtr(true),
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err == nil {
		t.Error("expected error with 33 results, got success")
	}

	// The error surfaces as "assertions failed" because the too-many-results error
	// is captured per-assertion. The assertion name should be in the failed list.
	if !strings.Contains(err.Error(), "items_exist") {
		t.Errorf("expected failed assertion 'items_exist' in error, got: %v", err)
	}

	if *result != types.TaskResultFailure {
		t.Errorf("result = %v, want TaskResultFailure", *result)
	}
}

func TestTask_NestedObjectEquality(t *testing.T) {
	// Test that nested objects with float64 vs int are correctly compared
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// JSON numbers decode as float64
		_, _ = w.Write([]byte(`{"service": {"latency_ms": 5, "status": "ok"}}`))
	}))
	defer server.Close()

	cfg := DefaultConfig()
	cfg.URL = server.URL
	cfg.Assertions = []AssertionConfig{
		{
			Name:     "service_check",
			Query:    ".service",
			Operator: OperatorEq,
			// YAML would decode latency_ms as int
			Value: map[string]any{
				testLatencyMS: 5, // int, not float64
				testStatus:    testStatusOK,
			},
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	task, result := newTestTaskWithContext(&cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := task.Execute(ctx)
	if err != nil {
		t.Errorf("expected success with nested object equality, got error: %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", *result)
	}
}

func TestExecuteJQQuery_Timeout(t *testing.T) {
	// Test that executeJQQuery respects context cancellation
	cfg := DefaultConfig()
	cfg.URL = "http://example.com" // Won't be used
	cfg.Assertions = []AssertionConfig{
		{
			Name:     "test",
			Query:    ".value",
			Operator: OperatorEq,
			Value:    42,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	task, _ := newTestTaskWithContext(&cfg)

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	jsonData := map[string]any{testValueKey: float64(42)}

	// Execute the query with the cancelled context
	_, err := task.executeJQQuery(ctx, &cfg.Assertions[0], jsonData)
	if err == nil {
		t.Error("expected error with cancelled context, got nil")
		return
	}

	// The error should mention timeout
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected timeout or canceled error, got: %v", err)
	}
}
