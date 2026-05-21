package checkhttpmetrics

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	dto "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	testMetricsURL         = "http://localhost:9090/metrics"
	testMetric1            = "metric1"
	testLabelFoo           = "foo"
	testLabelBar           = "bar"
	testLabelEnv           = "env"
	testLabelRegion        = "region"
	testValueProd          = "prod"
	testAssertionName      = "test"
	testAssertionName2     = "test_assertion"
	testInvalidOpLabel     = "invalid operator"
	testResetName          = "reset_test"
	testCounterMetric      = "test_counter"
	testCheckCounterAssert = "check_counter"
)

// validBaseConfig returns a config with required fields set for validation to pass intervals check.
func validBaseConfig() Config {
	return Config{
		PollInterval:   helper.Duration{Duration: 10 * time.Second},
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
	}
}

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
			name: "missing assertions",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL

				return c
			},
			wantErr: "at least one assertion is required",
		},
		{
			name: "missing assertion name",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.Assertions = []AssertionConfig{
					{Metric: "test_metric", Operator: OperatorGt, Value: 0},
				}

				return c
			},
			wantErr: "assertion[0]: name is required",
		},
		{
			name: "duplicate assertion names",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Metric: testMetric1, Operator: OperatorGt, Value: 0},
					{Name: testAssertionName, Metric: "metric2", Operator: OperatorGt, Value: 0},
				}

				return c
			},
			wantErr: "assertion[1]: duplicate name",
		},
		{
			name: testInvalidOpLabel,
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Metric: testMetric1, Operator: "invalid", Value: 0},
				}

				return c
			},
			wantErr: testInvalidOpLabel,
		},
		{
			name: "missing operator",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Metric: testMetric1, Value: 0},
				}

				return c
			},
			wantErr: "operator is required",
		},
		{
			name: "invalid max response size",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.MaxResponseSize = "invalid"
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Metric: testMetric1, Operator: OperatorGt, Value: 0},
				}

				return c
			},
			wantErr: "invalid maxResponseSize",
		},
		{
			name: "zero max response size",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.MaxResponseSize = "0B"
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Metric: testMetric1, Operator: OperatorGt, Value: 0},
				}

				return c
			},
			wantErr: "maxResponseSize must be positive",
		},
		{
			name: "valid config",
			configFunc: func() Config {
				c := validBaseConfig()
				c.URL = testMetricsURL
				c.MaxResponseSize = "10MB"
				c.MissingMetric = MissingBehaviorWait
				c.MissingSeries = MissingBehaviorFail
				c.ResetBehavior = ResetBehaviorFail
				c.Assertions = []AssertionConfig{
					{Name: testAssertionName, Metric: testMetric1, Operator: OperatorGt, Value: 0},
				}

				return c
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.configFunc()
			err := cfg.Validate()

			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestMatchLabels(t *testing.T) {
	tests := []struct {
		name         string
		metricLabels []*dto.LabelPair
		wantLabels   map[string]string
		want         bool
	}{
		{
			name:         "empty want labels matches anything",
			metricLabels: []*dto.LabelPair{{Name: proto.String(testLabelFoo), Value: proto.String(testLabelBar)}},
			wantLabels:   nil,
			want:         true,
		},
		{
			name:         "exact match",
			metricLabels: []*dto.LabelPair{{Name: proto.String(testLabelFoo), Value: proto.String(testLabelBar)}},
			wantLabels:   map[string]string{testLabelFoo: testLabelBar},
			want:         true,
		},
		{
			name: "partial match (subset of labels)",
			metricLabels: []*dto.LabelPair{
				{Name: proto.String(testLabelFoo), Value: proto.String(testLabelBar)},
				{Name: proto.String("baz"), Value: proto.String("qux")},
			},
			wantLabels: map[string]string{testLabelFoo: testLabelBar},
			want:       true,
		},
		{
			name:         "no match - different value",
			metricLabels: []*dto.LabelPair{{Name: proto.String(testLabelFoo), Value: proto.String(testLabelBar)}},
			wantLabels:   map[string]string{testLabelFoo: "baz"},
			want:         false,
		},
		{
			name:         "no match - missing label",
			metricLabels: []*dto.LabelPair{{Name: proto.String(testLabelFoo), Value: proto.String(testLabelBar)}},
			wantLabels:   map[string]string{"missing": "label"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchLabels(tt.metricLabels, tt.wantLabels)
			if got != tt.want {
				t.Errorf("matchLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateOperator(t *testing.T) {
	tests := []struct {
		name     string
		op       Operator
		actual   float64
		expected float64
		want     bool
	}{
		{"eq true", OperatorEq, 5, 5, true},
		{"eq false", OperatorEq, 5, 6, false},
		{"neq true", OperatorNeq, 5, 6, true},
		{"neq false", OperatorNeq, 5, 5, false},
		{"gt true", OperatorGt, 6, 5, true},
		{"gt false", OperatorGt, 5, 5, false},
		{"gte true equal", OperatorGte, 5, 5, true},
		{"gte true greater", OperatorGte, 6, 5, true},
		{"gte false", OperatorGte, 4, 5, false},
		{"lt true", OperatorLt, 4, 5, true},
		{"lt false", OperatorLt, 5, 5, false},
		{"lte true equal", OperatorLte, 5, 5, true},
		{"lte true less", OperatorLte, 4, 5, true},
		{"lte false", OperatorLte, 6, 5, false},
		{testInvalidOpLabel, Operator("invalid"), 5, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluateOperator(tt.op, tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("evaluateOperator(%v, %v, %v) = %v, want %v", tt.op, tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}

func TestGetMetricValue(t *testing.T) {
	counterVal := 42.0
	gaugeVal := 3.14
	untypedVal := 99.9
	summarySum := 100.0
	histogramSum := 200.0

	tests := []struct {
		name    string
		metric  *dto.Metric
		mType   dto.MetricType
		want    float64
		wantErr bool
	}{
		{
			name:   "counter",
			metric: &dto.Metric{Counter: &dto.Counter{Value: &counterVal}},
			mType:  dto.MetricType_COUNTER,
			want:   42.0,
		},
		{
			name:   "gauge",
			metric: &dto.Metric{Gauge: &dto.Gauge{Value: &gaugeVal}},
			mType:  dto.MetricType_GAUGE,
			want:   3.14,
		},
		{
			name:   "untyped",
			metric: &dto.Metric{Untyped: &dto.Untyped{Value: &untypedVal}},
			mType:  dto.MetricType_UNTYPED,
			want:   99.9,
		},
		{
			name:   "summary returns sample sum",
			metric: &dto.Metric{Summary: &dto.Summary{SampleSum: &summarySum}},
			mType:  dto.MetricType_SUMMARY,
			want:   100.0,
		},
		{
			name:   "histogram returns sample sum",
			metric: &dto.Metric{Histogram: &dto.Histogram{SampleSum: &histogramSum}},
			mType:  dto.MetricType_HISTOGRAM,
			want:   200.0,
		},
		{
			name:   "fallback to counter when type unknown",
			metric: &dto.Metric{Counter: &dto.Counter{Value: &counterVal}},
			mType:  dto.MetricType(-1),
			want:   42.0,
		},
		{
			name:    "empty metric returns error",
			metric:  &dto.Metric{},
			mType:   dto.MetricType_COUNTER,
			wantErr: true,
		},
		{
			name:    "type mismatch with no fallback returns error",
			metric:  &dto.Metric{},
			mType:   dto.MetricType_GAUGE,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMetricValue(tt.metric, tt.mType)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("getMetricValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MissingMetric != MissingBehaviorWait {
		t.Errorf("MissingMetric = %v, want %v", cfg.MissingMetric, MissingBehaviorWait)
	}

	if cfg.MissingSeries != MissingBehaviorWait {
		t.Errorf("MissingSeries = %v, want %v", cfg.MissingSeries, MissingBehaviorWait)
	}

	if cfg.ResetBehavior != ResetBehaviorFail {
		t.Errorf("ResetBehavior = %v, want %v", cfg.ResetBehavior, ResetBehaviorFail)
	}

	if cfg.MaxResponseSize != "10MB" {
		t.Errorf("MaxResponseSize = %v, want 10MB", cfg.MaxResponseSize)
	}
}

func TestGetMaxResponseSizeBytes(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want int64
	}{
		{
			name: "default when not set",
			cfg:  Config{},
			want: 10 * 1024 * 1024,
		},
		{
			name: "parsed value",
			cfg:  Config{MaxResponseSize: "5MB", maxResponseSizeBytes: 5 * 1024 * 1024},
			want: 5 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.GetMaxResponseSizeBytes()
			if got != tt.want {
				t.Errorf("GetMaxResponseSizeBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateMissingBehavior(t *testing.T) {
	validCases := []MissingBehavior{MissingBehaviorWait, MissingBehaviorFail, MissingBehaviorPass, ""}
	for _, v := range validCases {
		if err := validateMissingBehavior(v); err != nil {
			t.Errorf("validateMissingBehavior(%q) returned unexpected error: %v", v, err)
		}
	}

	if err := validateMissingBehavior("invalid"); err == nil {
		t.Error("validateMissingBehavior(\"invalid\") expected error, got nil")
	}
}

func TestValidateResetBehavior(t *testing.T) {
	validCases := []ResetBehavior{ResetBehaviorFail, ResetBehaviorRebaseline, ResetBehaviorIgnore, ""}
	for _, v := range validCases {
		if err := validateResetBehavior(v); err != nil {
			t.Errorf("validateResetBehavior(%q) returned unexpected error: %v", v, err)
		}
	}

	if err := validateResetBehavior("invalid"); err == nil {
		t.Error("validateResetBehavior(\"invalid\") expected error, got nil")
	}
}

func TestValidateMode(t *testing.T) {
	validCases := []AssertionMode{AssertionModeValue, AssertionModeDelta, ""}
	for _, v := range validCases {
		if err := validateMode(v); err != nil {
			t.Errorf("validateMode(%q) returned unexpected error: %v", v, err)
		}
	}

	if err := validateMode("invalid"); err == nil {
		t.Error("validateMode(\"invalid\") expected error, got nil")
	}
}

func TestValidateOperator(t *testing.T) {
	validCases := []Operator{OperatorEq, OperatorNeq, OperatorGt, OperatorGte, OperatorLt, OperatorLte}
	for _, v := range validCases {
		if err := validateOperator(v); err != nil {
			t.Errorf("validateOperator(%q) returned unexpected error: %v", v, err)
		}
	}

	if err := validateOperator(""); err == nil {
		t.Error("validateOperator(\"\") expected error, got nil")
	}

	if err := validateOperator("invalid"); err == nil {
		t.Error("validateOperator(\"invalid\") expected error, got nil")
	}
}

// newTestTask creates a Task for testing with a no-op logger.
func newTestTask(cfg *Config) *Task {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	c := Config{}
	if cfg != nil {
		c = *cfg
	}

	return &Task{
		config:    c,
		logger:    logger,
		baselines: make(map[string]float64),
	}
}

// makeCounterFamily creates a COUNTER metric family with the given metrics.
func makeCounterFamily(_ string, metrics ...*dto.Metric) *dto.MetricFamily {
	mType := dto.MetricType_COUNTER

	return &dto.MetricFamily{
		Name:   proto.String(testMetric1),
		Type:   &mType,
		Metric: metrics,
	}
}

// makeGaugeFamily creates a GAUGE metric family with the given metrics.
func makeGaugeFamily(name string, metrics ...*dto.Metric) *dto.MetricFamily {
	mType := dto.MetricType_GAUGE

	return &dto.MetricFamily{
		Name:   proto.String(name),
		Type:   &mType,
		Metric: metrics,
	}
}

// makeCounter creates a counter metric with labels and value.
func makeCounter(value float64, labels map[string]string) *dto.Metric {
	m := &dto.Metric{
		Counter: &dto.Counter{Value: proto.Float64(value)},
	}

	for k, v := range labels {
		m.Label = append(m.Label, &dto.LabelPair{
			Name:  proto.String(k),
			Value: proto.String(v),
		})
	}

	return m
}

// makeGauge creates a gauge metric with labels and value.
func makeGauge(value float64, labels map[string]string) *dto.Metric {
	m := &dto.Metric{
		Gauge: &dto.Gauge{Value: proto.Float64(value)},
	}

	for k, v := range labels {
		m.Label = append(m.Label, &dto.LabelPair{
			Name:  proto.String(k),
			Value: proto.String(v),
		})
	}

	return m
}

// makeUntypedFamily creates an UNTYPED metric family with the given metrics.
func makeUntypedFamily(name string, metrics ...*dto.Metric) *dto.MetricFamily {
	mType := dto.MetricType_UNTYPED

	return &dto.MetricFamily{
		Name:   proto.String(name),
		Type:   &mType,
		Metric: metrics,
	}
}

// makeUntyped creates an untyped metric with labels and value.
func makeUntyped(value float64, labels map[string]string) *dto.Metric {
	m := &dto.Metric{
		Untyped: &dto.Untyped{Value: proto.Float64(value)},
	}

	for k, v := range labels {
		m.Label = append(m.Label, &dto.LabelPair{
			Name:  proto.String(k),
			Value: proto.String(v),
		})
	}

	return m
}

func TestFindMetricValue(t *testing.T) {
	tests := []struct {
		name      string
		mf        *dto.MetricFamily
		labels    map[string]string
		wantValue float64
		wantErr   string
	}{
		{
			name:      "single series no labels",
			mf:        makeCounterFamily(testMetric1, makeCounter(42, nil)),
			labels:    nil,
			wantValue: 42,
		},
		{
			name: "single series with matching labels",
			mf: makeCounterFamily(testMetric1, makeCounter(100, map[string]string{
				testLabelEnv: testValueProd,
			})),
			labels:    map[string]string{testLabelEnv: testValueProd},
			wantValue: 100,
		},
		{
			name:    "no matching series",
			mf:      makeCounterFamily(testMetric1, makeCounter(42, map[string]string{testLabelEnv: testValueProd})),
			labels:  map[string]string{testLabelEnv: "dev"},
			wantErr: "no matching series",
		},
		{
			name: "multiple matching series error",
			mf: makeCounterFamily(testMetric1,
				makeCounter(10, map[string]string{testLabelEnv: testValueProd}),
				makeCounter(20, map[string]string{testLabelEnv: testValueProd}),
			),
			labels:  map[string]string{testLabelEnv: testValueProd},
			wantErr: "labels match 2 series",
		},
		{
			name: "partial label match selects correct series",
			mf: makeCounterFamily(testMetric1,
				makeCounter(10, map[string]string{testLabelEnv: testValueProd, testLabelRegion: "us"}),
				makeCounter(20, map[string]string{testLabelEnv: testValueProd, testLabelRegion: "eu"}),
			),
			labels:    map[string]string{testLabelEnv: testValueProd, testLabelRegion: "eu"},
			wantValue: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestTask(&Config{})
			val, err := task.findMetricValue(tt.mf, tt.labels)

			switch {
			case tt.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
			case tt.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.wantErr == "" && val != tt.wantValue:
				t.Errorf("findMetricValue() = %v, want %v", val, tt.wantValue)
			}
		})
	}
}

func TestHandleMissing(t *testing.T) {
	tests := []struct {
		name     string
		behavior MissingBehavior
		wantErr  bool
		wantPass bool
		wantWait bool
	}{
		{"fail behavior", MissingBehaviorFail, true, false, false},
		{"pass behavior", MissingBehaviorPass, false, true, false},
		{"wait behavior", MissingBehaviorWait, false, false, true},
		{"empty defaults to wait", "", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestTask(&Config{})
			result := assertionResult{name: "test"}
			result = task.handleMissing(result, tt.behavior, "metric not found")

			if tt.wantErr && result.err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && result.err != nil {
				t.Errorf("unexpected error: %v", result.err)
			}

			if result.passed != tt.wantPass {
				t.Errorf("passed = %v, want %v", result.passed, tt.wantPass)
			}

			if result.waiting != tt.wantWait {
				t.Errorf("waiting = %v, want %v", result.waiting, tt.wantWait)
			}
		})
	}
}

func TestEvaluateAssertion_MissingMetric(t *testing.T) {
	tests := []struct {
		name     string
		behavior MissingBehavior
		wantErr  bool
		wantPass bool
		wantWait bool
	}{
		{"wait on missing metric", MissingBehaviorWait, false, false, true},
		{"fail on missing metric", MissingBehaviorFail, true, false, false},
		{"pass on missing metric", MissingBehaviorPass, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestTask(&Config{MissingMetric: tt.behavior})
			assertion := &AssertionConfig{
				Name:     testAssertionName2,
				Metric:   "nonexistent_metric",
				Operator: OperatorGt,
				Value:    0,
			}

			// Empty metric families - metric doesn't exist
			result := task.evaluateAssertion(assertion, map[string]*dto.MetricFamily{})

			if tt.wantErr {
				if result.err == nil {
					t.Fatal("expected error, got nil")
				}

				if !strings.Contains(result.err.Error(), "not found") {
					t.Errorf("error %q does not contain 'not found'", result.err.Error())
				}
			} else if result.err != nil {
				t.Fatalf("unexpected error: %v", result.err)
			}

			if result.passed != tt.wantPass {
				t.Errorf("passed = %v, want %v", result.passed, tt.wantPass)
			}

			if result.waiting != tt.wantWait {
				t.Errorf("waiting = %v, want %v", result.waiting, tt.wantWait)
			}
		})
	}
}

func TestEvaluateAssertion_MissingSeries(t *testing.T) {
	tests := []struct {
		name     string
		behavior MissingBehavior
		wantErr  bool
		wantPass bool
		wantWait bool
	}{
		{"wait on missing series", MissingBehaviorWait, false, false, true},
		{"fail on missing series", MissingBehaviorFail, true, false, false},
		{"pass on missing series", MissingBehaviorPass, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestTask(&Config{MissingSeries: tt.behavior})
			assertion := &AssertionConfig{
				Name:     testAssertionName2,
				Metric:   testMetric1,
				Labels:   map[string]string{testLabelEnv: "nonexistent"},
				Operator: OperatorGt,
				Value:    0,
			}

			// Metric exists but labels don't match
			families := map[string]*dto.MetricFamily{
				testMetric1: makeCounterFamily(testMetric1, makeCounter(42, map[string]string{testLabelEnv: testValueProd})),
			}
			result := task.evaluateAssertion(assertion, families)

			if tt.wantErr {
				if result.err == nil {
					t.Fatal("expected error, got nil")
				}

				if !strings.Contains(result.err.Error(), "no matching series") {
					t.Errorf("error %q does not contain 'no matching series'", result.err.Error())
				}
			} else if result.err != nil {
				t.Fatalf("unexpected error: %v", result.err)
			}

			if result.passed != tt.wantPass {
				t.Errorf("passed = %v, want %v", result.passed, tt.wantPass)
			}

			if result.waiting != tt.wantWait {
				t.Errorf("waiting = %v, want %v", result.waiting, tt.wantWait)
			}
		})
	}
}

func TestEvaluateAssertion_AssertionOverridesBehavior(t *testing.T) {
	// Global behavior is wait, but assertion overrides to fail
	failBehavior := MissingBehaviorFail
	task := newTestTask(&Config{MissingMetric: MissingBehaviorWait})
	assertion := &AssertionConfig{
		Name:          "testAssertionName2",
		Metric:        "nonexistent_metric",
		Operator:      OperatorGt,
		Value:         0,
		MissingMetric: &failBehavior,
	}

	result := task.evaluateAssertion(assertion, map[string]*dto.MetricFamily{})

	if result.err == nil {
		t.Fatal("expected error, got nil")
	}

	if result.waiting {
		t.Error("expected waiting=false when override is fail")
	}
}

func TestEvaluateAssertion_ValueMode(t *testing.T) {
	tests := []struct {
		name       string
		metricVal  float64
		operator   Operator
		assertVal  float64
		wantPassed bool
	}{
		{"gt passes", 100, OperatorGt, 50, true},
		{"gt fails", 50, OperatorGt, 100, false},
		{"eq passes", 42, OperatorEq, 42, true},
		{"eq fails", 42, OperatorEq, 43, false},
		{"lte passes", 50, OperatorLte, 50, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := newTestTask(&Config{})
			assertion := &AssertionConfig{
				Name:     testAssertionName2,
				Metric:   testMetric1,
				Mode:     AssertionModeValue,
				Operator: tt.operator,
				Value:    tt.assertVal,
			}

			families := map[string]*dto.MetricFamily{
				testMetric1: makeCounterFamily(testMetric1, makeCounter(tt.metricVal, nil)),
			}
			result := task.evaluateAssertion(assertion, families)

			if result.err != nil {
				t.Fatalf("unexpected error: %v", result.err)
			}

			if result.waiting {
				t.Error("unexpected waiting=true")
			}

			if result.passed != tt.wantPassed {
				t.Errorf("passed = %v, want %v", result.passed, tt.wantPassed)
			}

			if result.value != tt.metricVal {
				t.Errorf("value = %v, want %v", result.value, tt.metricVal)
			}

			if !math.IsNaN(result.delta) {
				t.Errorf("delta = %v, want NaN", result.delta)
			}
		})
	}
}

func TestEvaluateAssertion_DeltaMode_FirstScrape(t *testing.T) {
	task := newTestTask(&Config{})
	assertion := &AssertionConfig{
		Name:     "delta_test",
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorGt,
		Value:    10,
	}

	families := map[string]*dto.MetricFamily{
		testMetric1: makeCounterFamily(testMetric1, makeCounter(100, nil)),
	}

	// First scrape should record baseline and wait
	result := task.evaluateAssertion(assertion, families)

	if !result.waiting {
		t.Error("first scrape should wait")
	}

	if result.passed {
		t.Error("first scrape should not pass yet")
	}

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if task.baselines["delta_test"] != 100.0 {
		t.Errorf("baseline = %v, want 100.0", task.baselines["delta_test"])
	}
}

func TestEvaluateAssertion_DeltaMode_SecondScrape(t *testing.T) {
	task := newTestTask(&Config{})
	task.baselines["delta_test"] = 100.0 // Simulate first scrape

	assertion := &AssertionConfig{
		Name:     "delta_test",
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorGt,
		Value:    10,
	}

	families := map[string]*dto.MetricFamily{
		testMetric1: makeCounterFamily(testMetric1, makeCounter(150, nil)),
	}

	// Second scrape should evaluate delta (150 - 100 = 50)
	result := task.evaluateAssertion(assertion, families)

	if result.waiting {
		t.Error("second scrape should not wait")
	}

	if !result.passed {
		t.Error("delta 50 > 10 should pass")
	}

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if result.value != 150.0 {
		t.Errorf("value = %v, want 150.0", result.value)
	}

	if result.delta != 50.0 {
		t.Errorf("delta = %v, want 50.0", result.delta)
	}
}

func TestEvaluateAssertion_CounterReset_Fail(t *testing.T) {
	task := newTestTask(&Config{ResetBehavior: ResetBehaviorFail})
	task.baselines[testResetName] = 100.0

	assertion := &AssertionConfig{
		Name:     testResetName,
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorGte,
		Value:    0,
	}

	// Counter dropped from 100 to 50 (reset detected)
	families := map[string]*dto.MetricFamily{
		testMetric1: makeCounterFamily(testMetric1, makeCounter(50, nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	if result.err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(result.err.Error(), "counter reset detected") {
		t.Errorf("error %q does not contain 'counter reset detected'", result.err.Error())
	}
}

func TestEvaluateAssertion_CounterReset_Rebaseline(t *testing.T) {
	task := newTestTask(&Config{ResetBehavior: ResetBehaviorRebaseline})
	task.baselines[testResetName] = 100.0

	assertion := &AssertionConfig{
		Name:     testResetName,
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorGte,
		Value:    0,
	}

	// Counter dropped from 100 to 50 (reset detected)
	families := map[string]*dto.MetricFamily{
		testMetric1: makeCounterFamily(testMetric1, makeCounter(50, nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if !result.waiting {
		t.Error("should wait after rebaseline")
	}

	if task.baselines[testResetName] != 50.0 {
		t.Errorf("baseline = %v, want 50.0", task.baselines[testResetName])
	}
}

func TestEvaluateAssertion_CounterReset_Ignore(t *testing.T) {
	task := newTestTask(&Config{ResetBehavior: ResetBehaviorIgnore})
	task.baselines[testResetName] = 100.0

	assertion := &AssertionConfig{
		Name:     testResetName,
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorLt,
		Value:    0,
	}

	// Counter dropped from 100 to 50 (reset detected but ignored)
	families := map[string]*dto.MetricFamily{
		testMetric1: makeCounterFamily(testMetric1, makeCounter(50, nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if result.waiting {
		t.Error("should not wait when ignoring resets")
	}

	// Delta = 50 - 100 = -50, which is < 0
	if !result.passed {
		t.Error("negative delta should be allowed when ignoring resets")
	}

	if result.delta != -50.0 {
		t.Errorf("delta = %v, want -50.0", result.delta)
	}
}

func TestEvaluateAssertion_GaugeDecrease_NoReset(t *testing.T) {
	// Gauges can decrease normally - this should NOT trigger reset detection
	task := newTestTask(&Config{ResetBehavior: ResetBehaviorFail})
	task.baselines["gauge_test"] = 100.0

	assertion := &AssertionConfig{
		Name:     "gauge_test",
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorLt,
		Value:    0,
	}

	// Gauge dropped from 100 to 50 - this is normal for gauges
	families := map[string]*dto.MetricFamily{
		testMetric1: makeGaugeFamily(testMetric1, makeGauge(50, nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	// Should NOT error even though ResetBehavior is Fail - gauges don't reset
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if result.waiting {
		t.Error("should not wait")
	}

	if !result.passed {
		t.Error("gauge decrease should work normally")
	}

	if result.delta != -50.0 {
		t.Errorf("delta = %v, want -50.0", result.delta)
	}
}

func TestEvaluateAssertion_UntypedDecrease_NoReset(t *testing.T) {
	// UNTYPED metrics (metrics without a TYPE declaration) should NOT trigger reset detection
	task := newTestTask(&Config{ResetBehavior: ResetBehaviorFail})
	task.baselines["untyped_test"] = 100.0

	assertion := &AssertionConfig{
		Name:     "untyped_test",
		Metric:   testMetric1,
		Mode:     AssertionModeDelta,
		Operator: OperatorLt,
		Value:    0,
	}

	// Untyped metric dropped from 100 to 50 - should NOT trigger reset detection
	families := map[string]*dto.MetricFamily{
		testMetric1: makeUntypedFamily(testMetric1, makeUntyped(50, nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	// Should NOT error even though ResetBehavior is Fail - only COUNTER type triggers reset
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	if result.waiting {
		t.Error("should not wait")
	}

	if !result.passed {
		t.Error("untyped decrease should work normally without reset detection")
	}

	if result.delta != -50.0 {
		t.Errorf("delta = %v, want -50.0", result.delta)
	}
}

func TestEvaluateAssertion_NaN(t *testing.T) {
	task := newTestTask(&Config{})
	assertion := &AssertionConfig{
		Name:     "nan_test",
		Metric:   testMetric1,
		Operator: OperatorGt,
		Value:    0,
	}

	families := map[string]*dto.MetricFamily{
		testMetric1: makeGaugeFamily(testMetric1, makeGauge(math.NaN(), nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	if result.err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(result.err.Error(), "non-finite") {
		t.Errorf("error %q does not contain 'non-finite'", result.err.Error())
	}
}

func TestEvaluateAssertion_Inf(t *testing.T) {
	task := newTestTask(&Config{})
	assertion := &AssertionConfig{
		Name:     "inf_test",
		Metric:   testMetric1,
		Operator: OperatorGt,
		Value:    0,
	}

	families := map[string]*dto.MetricFamily{
		testMetric1: makeGaugeFamily(testMetric1, makeGauge(math.Inf(1), nil)),
	}

	result := task.evaluateAssertion(assertion, families)

	if result.err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(result.err.Error(), "non-finite") {
		t.Errorf("error %q does not contain 'non-finite'", result.err.Error())
	}
}

func TestScrapeMetrics(t *testing.T) {
	metricsBody := `# HELP test_counter A test counter
# TYPE test_counter counter
test_counter{env="prod"} 42
test_counter{env="dev"} 10
# HELP test_gauge A test gauge
# TYPE test_gauge gauge
test_gauge 3.14
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(metricsBody))
	}))
	defer server.Close()

	task := newTestTask(&Config{
		URL:            server.URL,
		RequestTimeout: helper.Duration{Duration: 5 * time.Second},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	families, err := task.scrapeMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := families[testCounterMetric]; !ok {
		t.Error("missing test_counter metric")
	}

	if _, ok := families["test_gauge"]; !ok {
		t.Error("missing test_gauge metric")
	}

	// Verify counter values
	counterFamily := families[testCounterMetric]
	if counterFamily.GetType() != dto.MetricType_COUNTER {
		t.Errorf("counter type = %v, want COUNTER", counterFamily.GetType())
	}

	if len(counterFamily.GetMetric()) != 2 {
		t.Errorf("counter metric count = %d, want 2", len(counterFamily.GetMetric()))
	}

	// Verify gauge value
	gaugeFamily := families["test_gauge"]
	if gaugeFamily.GetType() != dto.MetricType_GAUGE {
		t.Errorf("gauge type = %v, want GAUGE", gaugeFamily.GetType())
	}

	if len(gaugeFamily.GetMetric()) != 1 {
		t.Errorf("gauge metric count = %d, want 1", len(gaugeFamily.GetMetric()))
	}
}

func TestScrapeMetrics_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	task := newTestTask(&Config{URL: server.URL})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	_, err := task.scrapeMetrics(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q does not contain '500'", err.Error())
	}
}

func TestScrapeMetrics_ResponseTooLarge(t *testing.T) {
	// Create a response larger than the limit
	largeBody := make([]byte, 100)
	for i := range largeBody {
		largeBody[i] = 'x'
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(largeBody)
	}))
	defer server.Close()

	task := newTestTask(&Config{
		URL:                  server.URL,
		maxResponseSizeBytes: 50,
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	_, err := task.scrapeMetrics(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "exceeds max size") {
		t.Errorf("error %q does not contain 'exceeds max size'", err.Error())
	}
}

func TestScrapeMetrics_InvalidPrometheusFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not prometheus format {{{"))
	}))
	defer server.Close()

	task := newTestTask(&Config{URL: server.URL})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	_, err := task.scrapeMetrics(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error %q does not contain 'parse'", err.Error())
	}
}

func TestScrapeMetrics_CustomHeaders(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# HELP m A metric\n# TYPE m gauge\nm 1\n"))
	}))
	defer server.Close()

	task := newTestTask(&Config{
		URL:     server.URL,
		Headers: map[string]string{"Authorization": "Bearer token123"},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	_, err := task.scrapeMetrics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedAuth != "Bearer token123" {
		t.Errorf("Authorization header = %q, want 'Bearer token123'", receivedAuth)
	}
}

func newTestTaskWithContext(cfg *Config) (*Task, types.Variables, *types.TaskResult) {
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

	task := &Task{
		ctx:       ctx,
		config:    config,
		logger:    logger,
		baselines: make(map[string]float64),
	}

	return task, outputs, &lastResult
}

func TestRunCheck_ScrapeError_FailOnCheckMiss(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	task, outputs, result := newTestTaskWithContext(&Config{
		URL:             server.URL,
		FailOnCheckMiss: true,
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	done, err := task.runCheck(context.Background())

	if !done {
		t.Error("expected done=true")
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "scrape failed") {
		t.Errorf("error %q does not contain 'scrape failed'", err.Error())
	}

	if *result != types.TaskResultFailure {
		t.Errorf("result = %v, want TaskResultFailure", *result)
	}

	if outputs.GetVar("scrapeErrors") != 1 {
		t.Errorf("scrapeErrors = %v, want 1", outputs.GetVar("scrapeErrors"))
	}
}

func TestRunCheck_ScrapeError_RetryOnFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	task, outputs, result := newTestTaskWithContext(&Config{
		URL:             server.URL,
		FailOnCheckMiss: false,
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	done, err := task.runCheck(context.Background())

	if done {
		t.Error("expected done=false")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *result != types.TaskResultNone {
		t.Errorf("result = %v, want TaskResultNone", *result)
	}

	if outputs.GetVar("scrapeErrors") != 1 {
		t.Errorf("scrapeErrors = %v, want 1", outputs.GetVar("scrapeErrors"))
	}
}

func TestRunCheck_AllAssertionsPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# TYPE test_counter counter\ntest_counter 100\n"))
	}))
	defer server.Close()

	task, outputs, result := newTestTaskWithContext(&Config{
		URL:           server.URL,
		MissingMetric: MissingBehaviorFail,
		MissingSeries: MissingBehaviorFail,
		Assertions: []AssertionConfig{
			{Name: testCheckCounterAssert, Metric: testCounterMetric, Mode: AssertionModeValue, Operator: OperatorGte, Value: 50},
		},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	done, err := task.runCheck(context.Background())

	if !done {
		t.Error("expected done=true")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", *result)
	}

	passed, ok := outputs.GetVar("passedAssertions").([]interface{})
	if !ok {
		t.Fatalf("passedAssertions type = %T, want []interface{}", outputs.GetVar("passedAssertions"))
	}

	if len(passed) != 1 {
		t.Errorf("passedAssertions length = %d, want 1", len(passed))
	}

	if len(passed) > 0 && passed[0] != testCheckCounterAssert {
		t.Errorf("passedAssertions[0] = %v, want %q", passed[0], testCheckCounterAssert)
	}
}

func TestRunCheck_AssertionFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# TYPE test_counter counter\ntest_counter 10\n"))
	}))
	defer server.Close()

	task, outputs, result := newTestTaskWithContext(&Config{
		URL:             server.URL,
		FailOnCheckMiss: true,
		MissingMetric:   MissingBehaviorFail,
		MissingSeries:   MissingBehaviorFail,
		Assertions: []AssertionConfig{
			{Name: testCheckCounterAssert, Metric: testCounterMetric, Mode: AssertionModeValue, Operator: OperatorGte, Value: 50},
		},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	done, err := task.runCheck(context.Background())

	if !done {
		t.Error("expected done=true")
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if *result != types.TaskResultFailure {
		t.Errorf("result = %v, want TaskResultFailure", *result)
	}

	failed, ok := outputs.GetVar("failedAssertions").([]interface{})
	if !ok {
		t.Fatalf("failedAssertions type = %T, want []interface{}", outputs.GetVar("failedAssertions"))
	}

	if len(failed) != 1 {
		t.Errorf("failedAssertions length = %d, want 1", len(failed))
	}

	if len(failed) > 0 && failed[0] != testCheckCounterAssert {
		t.Errorf("failedAssertions[0] = %v, want %q", failed[0], testCheckCounterAssert)
	}
}

func TestRunCheck_MixedAssertions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# TYPE metric_a counter\nmetric_a 100\n# TYPE metric_b counter\nmetric_b 5\n"))
	}))
	defer server.Close()

	task, outputs, result := newTestTaskWithContext(&Config{
		URL:             server.URL,
		FailOnCheckMiss: false,
		MissingMetric:   MissingBehaviorWait,
		MissingSeries:   MissingBehaviorWait,
		Assertions: []AssertionConfig{
			{Name: "a_passes", Metric: "metric_a", Mode: AssertionModeValue, Operator: OperatorGte, Value: 50},
			{Name: "b_fails", Metric: "metric_b", Mode: AssertionModeValue, Operator: OperatorGte, Value: 50},
			{Name: "c_waits", Metric: "metric_missing", Mode: AssertionModeValue, Operator: OperatorGte, Value: 0},
		},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	done, err := task.runCheck(context.Background())

	// Should keep waiting because one assertion is waiting
	if done {
		t.Error("expected done=false")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *result != types.TaskResultNone {
		t.Errorf("result = %v, want TaskResultNone", *result)
	}

	passed, ok := outputs.GetVar("passedAssertions").([]interface{})
	if !ok {
		t.Fatalf("passedAssertions type = %T, want []interface{}", outputs.GetVar("passedAssertions"))
	}

	foundAPasses := false

	for _, p := range passed {
		if p == "a_passes" {
			foundAPasses = true
		}
	}

	if !foundAPasses {
		t.Error("passedAssertions should contain 'a_passes'")
	}

	failed, ok := outputs.GetVar("failedAssertions").([]interface{})
	if !ok {
		t.Fatalf("failedAssertions type = %T, want []interface{}", outputs.GetVar("failedAssertions"))
	}

	foundBFails := false

	for _, f := range failed {
		if f == "b_fails" {
			foundBFails = true
		}
	}

	if !foundBFails {
		t.Error("failedAssertions should contain 'b_fails'")
	}
}

func TestRunCheck_AssertionErrorIncrement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Metric with multiple matching series (causes assertion error)
		_, _ = w.Write([]byte("# TYPE m counter\nm{env=\"a\"} 1\nm{env=\"b\"} 2\n"))
	}))
	defer server.Close()

	task, outputs, _ := newTestTaskWithContext(&Config{
		URL:             server.URL,
		FailOnCheckMiss: false,
		MissingMetric:   MissingBehaviorFail,
		MissingSeries:   MissingBehaviorFail,
		Assertions: []AssertionConfig{
			{Name: "ambiguous", Metric: "m", Mode: AssertionModeValue, Operator: OperatorGte, Value: 0},
		},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	_, _ = task.runCheck(context.Background())

	if outputs.GetVar("assertionErrors") != 1 {
		t.Errorf("assertionErrors = %v, want 1", outputs.GetVar("assertionErrors"))
	}
}

func TestRunCheck_ContinueOnPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("# TYPE my_counter counter\nmy_counter 100\n"))
	}))
	defer server.Close()

	task, _, result := newTestTaskWithContext(&Config{
		URL:            server.URL,
		ContinueOnPass: true,
		MissingMetric:  MissingBehaviorFail,
		MissingSeries:  MissingBehaviorFail,
		Assertions: []AssertionConfig{
			{Name: "check", Metric: "my_counter", Mode: AssertionModeValue, Operator: OperatorGte, Value: 50},
		},
	})
	task.httpClient = &http.Client{Timeout: 5 * time.Second}

	done, err := task.runCheck(context.Background())

	// Should continue even though all assertions pass
	if done {
		t.Error("expected done=false with continueOnPass")
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if *result != types.TaskResultSuccess {
		t.Errorf("result = %v, want TaskResultSuccess", *result)
	}
}
