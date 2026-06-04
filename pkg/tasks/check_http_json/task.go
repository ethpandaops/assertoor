package checkhttpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/sirupsen/logrus"
)

const (
	outputTypeObject = "object"
	outputTypeInt    = "int"

	// jq execution limits
	maxResultsPerAssertion = 32
	queryTimeout           = 5 * time.Second
)

// Compile-time interface compliance check.
var _ types.Task = (*Task)(nil)

var (
	TaskName       = "check_http_json"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks HTTP JSON endpoint and evaluates assertions using jq queries.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "passedAssertions",
				Type:        "array",
				Description: "Array of assertion names that passed.",
			},
			{
				Name:        "failedAssertions",
				Type:        "array",
				Description: "Array of assertion names that failed.",
			},
			{
				Name:        "values",
				Type:        outputTypeObject,
				Description: "Map of assertion name to latest jq result.",
			},
			{
				Name:        "httpStatus",
				Type:        outputTypeInt,
				Description: "Latest HTTP status code.",
			},
			{
				Name:        "responseErrors",
				Type:        outputTypeInt,
				Description: "Number of HTTP/read/status errors.",
			},
			{
				Name:        "parseErrors",
				Type:        outputTypeInt,
				Description: "Number of JSON parse errors.",
			},
			{
				Name:        "assertionErrors",
				Type:        outputTypeInt,
				Description: "Number of assertion evaluation errors.",
			},
		},
		NewTask: NewTask,
	}
)

// Task implements the check_http_json task.
type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	httpClient *http.Client

	// State
	requestCount    int
	responseErrors  int
	parseErrors     int
	assertionErrors int
	lastHTTPStatus  int
}

// NewTask creates a new check_http_json task.
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
	for {
		t.requestCount++

		done, err := t.runCheck(ctx)
		if done {
			return err
		}

		select {
		case <-time.After(t.config.PollInterval.Duration):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// assertionResult holds the result of evaluating a single assertion.
type assertionResult struct {
	name    string
	passed  bool
	value   any
	err     error
	waiting bool
}

// checkOutputs holds the aggregated outputs from a check.
type checkOutputs struct {
	passedAssertions []string
	failedAssertions []string
	values           map[string]any
}

func (t *Task) runCheck(ctx context.Context) (bool, error) {
	// Make HTTP request
	statusCode, body, err := t.fetchJSON(ctx)
	t.lastHTTPStatus = statusCode

	if err != nil {
		t.responseErrors++
		t.logger.Warnf("Request error (attempt %d): %v", t.requestCount, err)

		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
			t.setOutputs(nil)

			return true, fmt.Errorf("request failed: %w", err)
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.setOutputs(nil)

		return false, nil
	}

	// Check status code
	if !t.config.IsExpectedStatus(statusCode) {
		t.responseErrors++
		t.logger.Warnf("Unexpected status %d (attempt %d)", statusCode, t.requestCount)

		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
			t.setOutputs(nil)

			return true, fmt.Errorf("unexpected status code: %d", statusCode)
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.setOutputs(nil)

		return false, nil
	}

	// Status-only check (no assertions)
	if len(t.config.Assertions) == 0 {
		t.logger.Infof("Status check passed: %d", statusCode)
		t.ctx.SetResult(types.TaskResultSuccess)
		t.setOutputs(&checkOutputs{
			passedAssertions: []string{},
			failedAssertions: []string{},
			values:           make(map[string]any, 0),
		})

		if t.config.ContinueOnPass {
			return false, nil
		}

		return true, nil
	}

	// Parse JSON body
	var jsonData any

	if err := json.Unmarshal(body, &jsonData); err != nil {
		t.parseErrors++
		t.logger.Warnf("JSON parse error (attempt %d): %v", t.requestCount, err)

		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
			t.setOutputs(nil)

			return true, fmt.Errorf("JSON parse error: %w", err)
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.setOutputs(nil)

		return false, nil
	}

	// Evaluate assertions
	results := make([]assertionResult, 0, len(t.config.Assertions))

	for i := range t.config.Assertions {
		result := t.evaluateAssertion(&t.config.Assertions[i], jsonData)
		results = append(results, result)

		switch {
		case result.err != nil:
			t.assertionErrors++
			t.logger.Warnf("Assertion %q error: %v", result.name, result.err)
		case result.passed:
			t.logger.Debugf("Assertion %q passed (value: %v)", result.name, result.value)
		case result.waiting:
			t.logger.Debugf("Assertion %q waiting", result.name)
		default:
			t.logger.Debugf("Assertion %q failed (value: %v)", result.name, result.value)
		}
	}

	// Aggregate results
	out := &checkOutputs{
		passedAssertions: make([]string, 0, len(results)),
		failedAssertions: make([]string, 0, len(results)),
		values:           make(map[string]any, len(results)),
	}

	hasWaiting := false
	hasFailed := false

	for _, r := range results {
		if r.value != nil {
			out.values[r.name] = r.value
		}

		switch {
		case r.err != nil:
			hasFailed = true

			out.failedAssertions = append(out.failedAssertions, r.name)
		case r.waiting:
			hasWaiting = true
		case r.passed:
			out.passedAssertions = append(out.passedAssertions, r.name)
		default:
			hasFailed = true

			out.failedAssertions = append(out.failedAssertions, r.name)
		}
	}

	t.setOutputs(out)

	// Determine task result
	if !hasWaiting && !hasFailed {
		t.ctx.SetResult(types.TaskResultSuccess)

		if t.config.ContinueOnPass {
			return false, nil
		}

		return true, nil
	}

	if hasFailed && t.config.FailOnCheckMiss {
		t.ctx.SetResult(types.TaskResultFailure)

		return true, fmt.Errorf("assertions failed: %v", out.failedAssertions)
	}

	t.ctx.SetResult(types.TaskResultNone)

	return false, nil
}

func (t *Task) fetchJSON(ctx context.Context) (statusCode int, body []byte, err error) {
	var bodyReader io.Reader
	if reqBody := t.config.GetEncodedBody(); len(reqBody) > 0 {
		bodyReader = bytes.NewReader(reqBody)
	}

	req, err := http.NewRequestWithContext(ctx, t.config.Method, t.config.URL, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range t.config.Headers {
		req.Header.Set(k, v)
	}

	// Set default Content-Type if body is present and Content-Type not set
	if len(t.config.GetEncodedBody()) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	// HEAD requests don't have a body
	if t.config.Method == MethodHead {
		return resp.StatusCode, nil, nil
	}

	// Read body with size limit
	maxSize := t.config.GetMaxResponseSizeBytes()
	limitedReader := io.LimitReader(resp.Body, maxSize+1)

	body, err = io.ReadAll(limitedReader)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if int64(len(body)) > maxSize {
		return resp.StatusCode, nil, fmt.Errorf("response body exceeds max size (%d bytes)", maxSize)
	}

	return resp.StatusCode, body, nil
}

func (t *Task) evaluateAssertion(assertion *AssertionConfig, jsonData any) assertionResult {
	result := assertionResult{name: assertion.Name}

	// Execute jq query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	queryResults, err := t.executeJQQuery(ctx, assertion, jsonData)
	if err != nil {
		result.err = err
		return result
	}

	// Handle exists mode
	if assertion.Exists != nil {
		hasNonNull := false

		for _, v := range queryResults {
			if v != nil {
				hasNonNull = true

				break
			}
		}

		result.passed = hasNonNull == *assertion.Exists

		if len(queryResults) > 0 {
			result.value = queryResults[0]
		}

		return result
	}

	// Handle comparison mode
	if len(queryResults) == 0 {
		// No results - handle missing
		return t.handleMissing(result, assertion)
	}

	if len(queryResults) > 1 {
		// Multiple results for scalar comparison
		result.err = fmt.Errorf("query returned %d results, scalar comparison requires exactly one", len(queryResults))
		return result
	}

	queryResult := queryResults[0]
	result.value = queryResult

	if queryResult == nil {
		// Null result - handle as missing
		return t.handleMissing(result, assertion)
	}

	// Evaluate comparison
	passed, err := evaluateOperator(assertion.Operator, queryResult, assertion.Value)
	if err != nil {
		result.err = err
		return result
	}

	result.passed = passed

	return result
}

func (t *Task) executeJQQuery(ctx context.Context, assertion *AssertionConfig, jsonData any) ([]any, error) {
	if assertion.compiledQuery == nil {
		return nil, fmt.Errorf("query not compiled")
	}

	results := make([]any, 0, maxResultsPerAssertion)
	iter := assertion.compiledQuery.RunWithContext(ctx, jsonData)

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if err, isErr := v.(error); isErr {
			// Check if the error is due to context cancellation
			if ctx.Err() != nil {
				return nil, fmt.Errorf("query timeout: %w", ctx.Err())
			}

			return nil, fmt.Errorf("jq error: %w", err)
		}

		results = append(results, v)

		if len(results) > maxResultsPerAssertion {
			return nil, fmt.Errorf("query returned too many results (max %d)", maxResultsPerAssertion)
		}
	}

	// Check context after iteration completes
	if ctx.Err() != nil {
		return nil, fmt.Errorf("query timeout: %w", ctx.Err())
	}

	return results, nil
}

func (t *Task) handleMissing(result assertionResult, assertion *AssertionConfig) assertionResult {
	// Check assertion-level allowMissing
	if assertion.AllowMissing != nil {
		if *assertion.AllowMissing {
			result.passed = true
		} else {
			result.err = fmt.Errorf("missing result and allowMissing is false")
		}

		return result
	}

	// Fall back to global failOnCheckMiss
	if t.config.FailOnCheckMiss {
		result.err = fmt.Errorf("missing result")
	} else {
		result.waiting = true
	}

	return result
}

func (t *Task) setOutputs(out *checkOutputs) {
	if out == nil {
		t.ctx.Outputs.SetVar("passedAssertions", []string{})
		t.ctx.Outputs.SetVar("failedAssertions", []string{})
		t.ctx.Outputs.SetVar("values", map[string]any{})
	} else {
		if data, err := vars.GeneralizeData(out.passedAssertions); err != nil {
			t.logger.Warnf("failed to generalize passedAssertions: %v", err)
		} else {
			t.ctx.Outputs.SetVar("passedAssertions", data)
		}

		if data, err := vars.GeneralizeData(out.failedAssertions); err != nil {
			t.logger.Warnf("failed to generalize failedAssertions: %v", err)
		} else {
			t.ctx.Outputs.SetVar("failedAssertions", data)
		}

		if data, err := vars.GeneralizeData(out.values); err != nil {
			t.logger.Warnf("failed to generalize values: %v", err)
		} else {
			t.ctx.Outputs.SetVar("values", data)
		}
	}

	t.ctx.Outputs.SetVar("httpStatus", t.lastHTTPStatus)
	t.ctx.Outputs.SetVar("responseErrors", t.responseErrors)
	t.ctx.Outputs.SetVar("parseErrors", t.parseErrors)
	t.ctx.Outputs.SetVar("assertionErrors", t.assertionErrors)
}

// evaluateOperator compares actual and expected values using the given operator.
func evaluateOperator(op Operator, actual, expected any) (bool, error) {
	switch op {
	case OperatorEq:
		return deepEqualWithNumericCoercion(actual, expected), nil

	case OperatorNeq:
		return !deepEqualWithNumericCoercion(actual, expected), nil

	case OperatorGt, OperatorGte, OperatorLt, OperatorLte:
		return compareNumeric(op, actual, expected)

	case OperatorContains:
		return evalContains(actual, expected)

	case OperatorNotContains:
		contains, err := evalContains(actual, expected)
		if err != nil {
			return false, err
		}

		return !contains, nil

	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

func compareNumeric(op Operator, actual, expected any) (bool, error) {
	actualNum, ok := toFloat64(actual)
	if !ok {
		return false, fmt.Errorf("actual value %v (%T) is not numeric", actual, actual)
	}

	expectedNum, ok := toFloat64(expected)
	if !ok {
		return false, fmt.Errorf("expected value %v (%T) is not numeric", expected, expected)
	}

	switch op {
	case OperatorGt:
		return actualNum > expectedNum, nil
	case OperatorGte:
		return actualNum >= expectedNum, nil
	case OperatorLt:
		return actualNum < expectedNum, nil
	case OperatorLte:
		return actualNum <= expectedNum, nil
	case OperatorEq, OperatorNeq, OperatorContains, OperatorNotContains:
		return false, fmt.Errorf("operator %s is not a numeric operator", op)
	default:
		return false, fmt.Errorf("unknown operator: %s", op)
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

// deepEqualWithNumericCoercion compares two values for equality, with special
// handling for numeric types. JSON parses numbers as float64, while YAML may
// parse them as int. This function ensures that numeric values are compared
// by value rather than by type (e.g., float64(42) equals int(42)).
// The comparison is recursive for maps and slices.
func deepEqualWithNumericCoercion(a, b any) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	// Try numeric comparison first
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)

	if aIsNum && bIsNum {
		return aNum == bNum
	}

	// If one is numeric and the other isn't, they're not equal
	if aIsNum != bIsNum {
		return false
	}

	// Handle maps recursively
	aMap, aIsMap := a.(map[string]any)
	bMap, bIsMap := b.(map[string]any)

	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}

		for k, av := range aMap {
			bv, exists := bMap[k]
			if !exists || !deepEqualWithNumericCoercion(av, bv) {
				return false
			}
		}

		return true
	}

	// Handle slices recursively
	aSlice, aIsSlice := a.([]any)
	bSlice, bIsSlice := b.([]any)

	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}

		for i := range aSlice {
			if !deepEqualWithNumericCoercion(aSlice[i], bSlice[i]) {
				return false
			}
		}

		return true
	}

	// Fall back to reflect.DeepEqual for other types (strings, bools, etc.)
	return reflect.DeepEqual(a, b)
}

func evalContains(actual, expected any) (bool, error) {
	switch a := actual.(type) {
	case string:
		e, ok := expected.(string)
		if !ok {
			return false, fmt.Errorf("contains: expected string for string comparison, got %T", expected)
		}

		return strings.Contains(a, e), nil

	case []any:
		for _, item := range a {
			if deepEqualWithNumericCoercion(item, expected) {
				return true, nil
			}
		}

		return false, nil

	case map[string]any:
		expectedMap, ok := expected.(map[string]any)
		if !ok {
			return false, fmt.Errorf("contains: expected object for object comparison, got %T", expected)
		}

		for k, v := range expectedMap {
			actualV, exists := a[k]
			if !exists || !deepEqualWithNumericCoercion(actualV, v) {
				return false, nil
			}
		}

		return true, nil

	default:
		return false, fmt.Errorf("contains: unsupported type %T", actual)
	}
}
