package checkhttpmetrics

import (
	"fmt"
	"math"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/ethpandaops/assertoor/pkg/helper"
)

// MissingBehavior controls what happens when a metric or series cannot be found.
// "wait" keeps polling, "fail" marks the assertion as failed, "pass" treats it as passed.
type MissingBehavior string

const (
	MissingBehaviorWait MissingBehavior = "wait"
	MissingBehaviorFail MissingBehavior = "fail"
	MissingBehaviorPass MissingBehavior = "pass"
)

// DefaultMaxResponseSize is the default maximum response body size for metrics scraping.
const DefaultMaxResponseSize = "10MB"

// ResetBehavior controls what happens when a counter value drops below its baseline,
// which typically indicates a service restart or counter reset.
type ResetBehavior string

const (
	ResetBehaviorFail       ResetBehavior = "fail"
	ResetBehaviorRebaseline ResetBehavior = "rebaseline"
	ResetBehaviorIgnore     ResetBehavior = "ignore"
)

// AssertionMode determines whether to compare the raw metric value or the change since baseline.
// Delta mode requires at least two scrapes: one to record baseline, one to evaluate.
type AssertionMode string

const (
	AssertionModeValue AssertionMode = "value"
	AssertionModeDelta AssertionMode = "delta"
)

// Operator specifies the comparison operation between actual and expected values.
type Operator string

const (
	OperatorEq  Operator = "eq"
	OperatorNeq Operator = "neq"
	OperatorGt  Operator = "gt"
	OperatorGte Operator = "gte"
	OperatorLt  Operator = "lt"
	OperatorLte Operator = "lte"
)

// AssertionConfig defines a single metric assertion to evaluate.
// Labels are a subset selector that must match exactly one time series;
// matching zero or multiple series is an error.
type AssertionConfig struct {
	Name          string            `yaml:"name" json:"name" desc:"Unique human-readable assertion name."`
	Metric        string            `yaml:"metric" json:"metric" desc:"Prometheus metric name."`
	Labels        map[string]string `yaml:"labels" json:"labels" desc:"Label selector; all specified labels must match and select exactly one time series."`
	Mode          AssertionMode     `yaml:"mode" json:"mode" desc:"Evaluation mode: 'value' (current value) or 'delta' (change since baseline)."`
	Operator      Operator          `yaml:"operator" json:"operator" desc:"Comparison operator: eq, neq, gt, gte, lt, lte."`
	Value         float64           `yaml:"value" json:"value" desc:"Expected value for comparison."`
	MissingMetric *MissingBehavior  `yaml:"missingMetric" json:"missingMetric,omitempty" desc:"Override global missingMetric behavior for this assertion."`
	MissingSeries *MissingBehavior  `yaml:"missingSeries" json:"missingSeries,omitempty" desc:"Override global missingSeries behavior for this assertion."`
}

// Config holds the task configuration for scraping a Prometheus metrics endpoint
// and evaluating assertions against the scraped values.
type Config struct {
	URL             string            `yaml:"url" json:"url" desc:"HTTP URL of the Prometheus metrics endpoint."`
	Headers         map[string]string `yaml:"headers" json:"headers" desc:"Optional HTTP request headers."`
	PollInterval    helper.Duration   `yaml:"pollInterval" json:"pollInterval" desc:"Interval between metric scrapes."`
	RequestTimeout  helper.Duration   `yaml:"requestTimeout" json:"requestTimeout" desc:"Timeout for a single HTTP request."`
	MaxResponseSize string            `yaml:"maxResponseSize" json:"maxResponseSize" desc:"Maximum response body size (e.g., '10MB')."`
	FailOnCheckMiss bool              `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail immediately when assertions are not met."`
	ContinueOnPass  bool              `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue checking after all assertions pass."`
	MissingMetric   MissingBehavior   `yaml:"missingMetric" json:"missingMetric" desc:"Behavior when metric doesn't exist: wait, fail, pass."`
	MissingSeries   MissingBehavior   `yaml:"missingSeries" json:"missingSeries" desc:"Behavior when no series matches labels: wait, fail, pass."`
	ResetBehavior   ResetBehavior     `yaml:"resetBehavior" json:"resetBehavior" desc:"Behavior on counter reset in delta mode: fail, rebaseline, ignore."`
	Assertions      []AssertionConfig `yaml:"assertions" json:"assertions" desc:"List of metric assertions to evaluate."`

	// Parsed values (not from YAML)
	maxResponseSizeBytes int64
}

func DefaultConfig() Config {
	return Config{
		PollInterval:    helper.Duration{Duration: 10 * time.Second},
		RequestTimeout:  helper.Duration{Duration: 5 * time.Second},
		MaxResponseSize: DefaultMaxResponseSize,
		MissingMetric:   MissingBehaviorWait,
		MissingSeries:   MissingBehaviorWait,
		ResetBehavior:   ResetBehaviorFail,
	}
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("url is required")
	}

	if len(c.Assertions) == 0 {
		return fmt.Errorf("at least one assertion is required")
	}

	// Validate intervals to prevent tight-loops or disabled timeouts
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

	// Validate assertions and check for duplicate names
	seenNames := make(map[string]bool, len(c.Assertions))

	for i, a := range c.Assertions {
		if a.Name == "" {
			return fmt.Errorf("assertion[%d]: name is required", i)
		}

		if seenNames[a.Name] {
			return fmt.Errorf("assertion[%d]: duplicate name %q", i, a.Name)
		}

		seenNames[a.Name] = true

		if a.Metric == "" {
			return fmt.Errorf("assertion[%d] %q: metric is required", i, a.Name)
		}

		if err := validateMode(a.Mode); err != nil {
			return fmt.Errorf("assertion[%d] %q: %w", i, a.Name, err)
		}

		if err := validateOperator(a.Operator); err != nil {
			return fmt.Errorf("assertion[%d] %q: %w", i, a.Name, err)
		}

		if a.MissingMetric != nil {
			if err := validateMissingBehavior(*a.MissingMetric); err != nil {
				return fmt.Errorf("assertion[%d] %q: missingMetric: %w", i, a.Name, err)
			}
		}

		if a.MissingSeries != nil {
			if err := validateMissingBehavior(*a.MissingSeries); err != nil {
				return fmt.Errorf("assertion[%d] %q: missingSeries: %w", i, a.Name, err)
			}
		}
	}

	// Validate global enums
	if err := validateMissingBehavior(c.MissingMetric); err != nil {
		return fmt.Errorf("missingMetric: %w", err)
	}

	if err := validateMissingBehavior(c.MissingSeries); err != nil {
		return fmt.Errorf("missingSeries: %w", err)
	}

	if err := validateResetBehavior(c.ResetBehavior); err != nil {
		return fmt.Errorf("resetBehavior: %w", err)
	}

	return nil
}

func (c *Config) GetMaxResponseSizeBytes() int64 {
	return c.maxResponseSizeBytes
}

func validateMissingBehavior(b MissingBehavior) error {
	switch b {
	case MissingBehaviorWait, MissingBehaviorFail, MissingBehaviorPass, "":
		return nil
	default:
		return fmt.Errorf("invalid value %q, must be one of: wait, fail, pass", b)
	}
}

func validateResetBehavior(b ResetBehavior) error {
	switch b {
	case ResetBehaviorFail, ResetBehaviorRebaseline, ResetBehaviorIgnore, "":
		return nil
	default:
		return fmt.Errorf("invalid value %q, must be one of: fail, rebaseline, ignore", b)
	}
}

func validateMode(m AssertionMode) error {
	switch m {
	case AssertionModeValue, AssertionModeDelta, "":
		return nil
	default:
		return fmt.Errorf("invalid mode %q, must be one of: value, delta", m)
	}
}

func validateOperator(o Operator) error {
	switch o {
	case OperatorEq, OperatorNeq, OperatorGt, OperatorGte, OperatorLt, OperatorLte:
		return nil
	case "":
		return fmt.Errorf("operator is required")
	default:
		return fmt.Errorf("invalid operator %q, must be one of: eq, neq, gt, gte, lt, lte", o)
	}
}
