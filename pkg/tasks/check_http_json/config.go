package checkhttpjson

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/itchyny/gojq"
)

const (
	// DefaultMaxResponseSize is the default maximum response body size.
	DefaultMaxResponseSize = "10MB"

	// MethodHead is the HEAD HTTP method constant.
	MethodHead = "HEAD"
)

// Operator specifies the comparison operation between actual and expected values.
type Operator string

const (
	OperatorEq          Operator = "eq"
	OperatorNeq         Operator = "neq"
	OperatorGt          Operator = "gt"
	OperatorGte         Operator = "gte"
	OperatorLt          Operator = "lt"
	OperatorLte         Operator = "lte"
	OperatorContains    Operator = "contains"
	OperatorNotContains Operator = "not_contains"
)

// allowedMethods lists HTTP methods allowed for this task.
var allowedMethods = map[string]bool{
	"GET":    true,
	"POST":   true,
	"PUT":    true,
	"PATCH":  true,
	"DELETE": true,
	"HEAD":   true,
}

// AssertionConfig defines a single JSON assertion to evaluate.
// Each assertion must use exactly one of: exists (for existence checks) or
// operator+value (for comparisons). Validation rejects assertions that set both or neither.
type AssertionConfig struct {
	Name         string   `yaml:"name" json:"name" desc:"Unique assertion name."`
	Query        string   `yaml:"query" json:"query" desc:"jq expression evaluated against the JSON response."`
	Exists       *bool    `yaml:"exists" json:"exists,omitempty" desc:"Assert whether the query returns at least one result."`
	Operator     Operator `yaml:"operator" json:"operator,omitempty" desc:"Comparison operator: eq, neq, gt, gte, lt, lte, contains, not_contains."`
	Value        any      `yaml:"value" json:"value,omitempty" desc:"Expected value for comparison."`
	AllowMissing *bool    `yaml:"allowMissing" json:"allowMissing,omitempty" desc:"Override missing result behavior for this assertion."`

	// Compiled jq query (not from YAML)
	compiledQuery *gojq.Code
}

// Config holds the task configuration for fetching JSON from an HTTP endpoint
// and evaluating assertions against the response.
type Config struct {
	URL             string            `yaml:"url" json:"url" desc:"HTTP URL of the JSON endpoint."`
	Method          string            `yaml:"method" json:"method" desc:"HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD)."`
	Headers         map[string]string `yaml:"headers" json:"headers" desc:"Optional HTTP request headers."`
	Body            any               `yaml:"body" json:"body,omitempty" desc:"Request body (YAML/JSON value, JSON-encoded before sending)."`
	BodyRaw         string            `yaml:"bodyRaw" json:"bodyRaw,omitempty" desc:"Raw request body (sent as-is, takes precedence over body)."`
	ExpectStatus    *int              `yaml:"expectStatus" json:"expectStatus,omitempty" desc:"Expected HTTP status code."`
	ExpectStatuses  []int             `yaml:"expectStatuses" json:"expectStatuses,omitempty" desc:"Multiple expected HTTP status codes."`
	PollInterval    helper.Duration   `yaml:"pollInterval" json:"pollInterval" desc:"Interval between requests."`
	RequestTimeout  helper.Duration   `yaml:"requestTimeout" json:"requestTimeout" desc:"Timeout for a single HTTP request."`
	MaxResponseSize string            `yaml:"maxResponseSize" json:"maxResponseSize" desc:"Maximum response body size (e.g., '10MB')."`
	FailOnCheckMiss bool              `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail immediately when assertions are not met."`
	ContinueOnPass  bool              `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue checking after all assertions pass."`
	Assertions      []AssertionConfig `yaml:"assertions" json:"assertions" desc:"List of JSON assertions to evaluate."`

	// Parsed values (not from YAML)
	maxResponseSizeBytes int64
	expectedStatuses     map[int]bool
	encodedBody          []byte
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Method:          "GET",
		PollInterval:    helper.Duration{Duration: 10 * time.Second},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		MaxResponseSize: DefaultMaxResponseSize,
	}
}

// Validate validates the configuration and compiles jq queries.
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	// Normalize and validate method
	c.Method = strings.ToUpper(c.Method)
	if !allowedMethods[c.Method] {
		return fmt.Errorf("invalid method %q, must be one of: GET, POST, PUT, PATCH, DELETE, HEAD", c.Method)
	}

	// HEAD cannot have assertions (no response body)
	if c.Method == MethodHead && len(c.Assertions) > 0 {
		return fmt.Errorf("HEAD requests cannot have assertions (no response body)")
	}

	// Validate status configuration
	if c.ExpectStatus != nil && len(c.ExpectStatuses) > 0 {
		return fmt.Errorf("cannot set both expectStatus and expectStatuses")
	}

	// Build expected statuses map
	c.expectedStatuses = make(map[int]bool, 8)

	switch {
	case c.ExpectStatus != nil:
		if !isValidHTTPStatus(*c.ExpectStatus) {
			return fmt.Errorf("invalid expectStatus %d: must be between 100 and 599", *c.ExpectStatus)
		}

		c.expectedStatuses[*c.ExpectStatus] = true
	case len(c.ExpectStatuses) > 0:
		for _, s := range c.ExpectStatuses {
			if !isValidHTTPStatus(s) {
				return fmt.Errorf("invalid expectStatuses value %d: must be between 100 and 599", s)
			}

			c.expectedStatuses[s] = true
		}
	default:
		c.expectedStatuses[200] = true
	}

	// Validate intervals
	if c.PollInterval.Duration <= 0 {
		return fmt.Errorf("pollInterval must be positive")
	}

	if c.RequestTimeout.Duration <= 0 {
		return fmt.Errorf("requestTimeout must be positive")
	}

	// Set default and parse max response size
	if c.MaxResponseSize == "" {
		c.MaxResponseSize = DefaultMaxResponseSize
	}

	size, err := humanize.ParseBytes(c.MaxResponseSize)
	if err != nil {
		return fmt.Errorf("invalid maxResponseSize %q: %w", c.MaxResponseSize, err)
	}

	if size == 0 {
		return fmt.Errorf("maxResponseSize must be positive")
	}

	if size > math.MaxInt64 {
		return fmt.Errorf("maxResponseSize %q exceeds maximum allowed value", c.MaxResponseSize)
	}

	c.maxResponseSizeBytes = int64(size)

	// Encode request body if present
	if c.BodyRaw != "" {
		c.encodedBody = []byte(c.BodyRaw)
	} else if c.Body != nil {
		encoded, err := json.Marshal(c.Body)
		if err != nil {
			return fmt.Errorf("failed to encode body as JSON: %w", err)
		}

		c.encodedBody = encoded
	}

	// Validate assertions
	if err := c.validateAssertions(); err != nil {
		return err
	}

	return nil
}

// validateAssertions validates all assertion configurations.
func (c *Config) validateAssertions() error {
	seenNames := make(map[string]bool, len(c.Assertions))

	for i := range c.Assertions {
		a := &c.Assertions[i]

		if err := c.validateAssertion(i, a, seenNames); err != nil {
			return err
		}

		seenNames[a.Name] = true
	}

	return nil
}

// validateAssertion validates a single assertion configuration.
func (c *Config) validateAssertion(idx int, a *AssertionConfig, seenNames map[string]bool) error {
	if a.Name == "" {
		return fmt.Errorf("assertion[%d]: name is required", idx)
	}

	if seenNames[a.Name] {
		return fmt.Errorf("assertion[%d]: duplicate name %q", idx, a.Name)
	}

	if a.Query == "" {
		return fmt.Errorf("assertion[%d] %q: query is required", idx, a.Name)
	}

	// Compile jq query
	query, err := gojq.Parse(a.Query)
	if err != nil {
		return fmt.Errorf("assertion[%d] %q: invalid jq syntax: %w", idx, a.Name, err)
	}

	code, err := gojq.Compile(query)
	if err != nil {
		return fmt.Errorf("assertion[%d] %q: failed to compile jq query: %w", idx, a.Name, err)
	}

	a.compiledQuery = code

	// Validate assertion mode: must have exactly one of exists or operator
	hasExists := a.Exists != nil
	hasOperator := a.Operator != ""

	if hasExists && hasOperator {
		return fmt.Errorf("assertion[%d] %q: cannot set both 'exists' and 'operator'", idx, a.Name)
	}

	if !hasExists && !hasOperator {
		return fmt.Errorf("assertion[%d] %q: must set either 'exists' or 'operator'", idx, a.Name)
	}

	// Validate operator if set
	if hasOperator {
		if err := validateOperator(a.Operator); err != nil {
			return fmt.Errorf("assertion[%d] %q: %w", idx, a.Name, err)
		}

		// Value is required when operator is set
		if a.Value == nil {
			return fmt.Errorf("assertion[%d] %q: 'value' is required when 'operator' is set", idx, a.Name)
		}
	}

	return nil
}

// GetMaxResponseSizeBytes returns the parsed max response size in bytes.
func (c *Config) GetMaxResponseSizeBytes() int64 {
	return c.maxResponseSizeBytes
}

// IsExpectedStatus checks if the given status code is expected.
func (c *Config) IsExpectedStatus(status int) bool {
	return c.expectedStatuses[status]
}

// GetEncodedBody returns the encoded request body.
func (c *Config) GetEncodedBody() []byte {
	return c.encodedBody
}

func validateOperator(o Operator) error {
	switch o {
	case OperatorEq, OperatorNeq, OperatorGt, OperatorGte, OperatorLt, OperatorLte,
		OperatorContains, OperatorNotContains:
		return nil
	default:
		return fmt.Errorf("invalid operator %q, must be one of: eq, neq, gt, gte, lt, lte, contains, not_contains", o)
	}
}

func isValidHTTPStatus(status int) bool {
	return status >= 100 && status <= 599
}
