package checkhttpmetrics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

const outputTypeObject = "object"

var (
	TaskName       = "check_http_metrics"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Checks HTTP Prometheus metrics endpoint and evaluates assertions against metric values.",
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
				Description: "Map of assertion name to latest observed value.",
			},
			{
				Name:        "deltas",
				Type:        outputTypeObject,
				Description: "Map of assertion name to computed delta for delta-mode assertions.",
			},
			{
				Name:        "baselines",
				Type:        outputTypeObject,
				Description: "Map of assertion name to baseline value for delta-mode assertions.",
			},
			{
				Name:        "scrapeErrors",
				Type:        "int",
				Description: "Number of HTTP/parsing errors encountered.",
			},
			{
				Name:        "assertionErrors",
				Type:        "int",
				Description: "Number of assertion evaluation errors.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	httpClient *http.Client

	// State
	baselines       map[string]float64
	scrapeCount     int
	scrapeErrors    int
	assertionErrors int
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:       ctx,
		options:   options,
		logger:    ctx.Logger.GetLogger(),
		baselines: make(map[string]float64, 8),
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

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	if err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars); err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	t.httpClient = &http.Client{
		Timeout: config.RequestTimeout.Duration,
	}

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	for {
		t.scrapeCount++

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

type assertionResult struct {
	name    string
	passed  bool
	value   float64
	delta   float64
	err     error
	waiting bool // true if we should keep waiting (missing metric/series with wait behavior)
}

func (t *Task) runCheck(ctx context.Context) (bool, error) {
	// Scrape metrics
	metricFamilies, err := t.scrapeMetrics(ctx)
	if err != nil {
		t.scrapeErrors++
		t.logger.Warnf("Scrape error (attempt %d): %v", t.scrapeCount, err)

		if t.config.FailOnCheckMiss {
			t.ctx.SetResult(types.TaskResultFailure)
			t.setOutputs(nil)

			return true, fmt.Errorf("scrape failed: %w", err)
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Scrape failed, retrying... (attempt %d)", t.scrapeCount))
		t.setOutputs(nil) // Update outputs so scrapeErrors is visible

		return false, nil
	}

	// Evaluate assertions
	results := t.evaluateAssertions(metricFamilies)

	// Collect results
	var passedAssertions, failedAssertions []string

	var anyWaiting, anyFailed bool

	values := make(map[string]float64, len(results))
	deltas := make(map[string]float64, len(results))

	for _, r := range results {
		if !math.IsNaN(r.value) {
			values[r.name] = r.value
		}

		if !math.IsNaN(r.delta) {
			deltas[r.name] = r.delta
		}

		if r.waiting {
			anyWaiting = true

			continue
		}

		if r.err != nil {
			t.assertionErrors++

			failedAssertions = append(failedAssertions, r.name)
			anyFailed = true

			t.logger.Warnf("Assertion %q error: %v", r.name, r.err)

			continue
		}

		if r.passed {
			passedAssertions = append(passedAssertions, r.name)
		} else {
			failedAssertions = append(failedAssertions, r.name)
			anyFailed = true
		}
	}

	t.setOutputs(&checkOutputs{
		passedAssertions: passedAssertions,
		failedAssertions: failedAssertions,
		values:           values,
		deltas:           deltas,
	})

	// Determine result
	allPassed := !anyFailed && !anyWaiting

	switch {
	case allPassed:
		t.ctx.SetResult(types.TaskResultSuccess)
		t.ctx.ReportProgress(100, fmt.Sprintf("All assertions passed (%d/%d)", len(passedAssertions), len(t.config.Assertions)))

		if !t.config.ContinueOnPass {
			return true, nil
		}

	case anyFailed && t.config.FailOnCheckMiss:
		t.ctx.SetResult(types.TaskResultFailure)

		return true, fmt.Errorf("assertions failed: %v", failedAssertions)

	default:
		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting... passed=%d failed=%d waiting=%d (attempt %d)",
			len(passedAssertions), len(failedAssertions), countWaiting(results), t.scrapeCount))
	}

	return false, nil
}

type checkOutputs struct {
	passedAssertions []string
	failedAssertions []string
	values           map[string]float64
	deltas           map[string]float64
}

func (t *Task) setOutputs(out *checkOutputs) {
	if out == nil {
		t.ctx.Outputs.SetVar("passedAssertions", []string{})
		t.ctx.Outputs.SetVar("failedAssertions", []string{})
		t.ctx.Outputs.SetVar("values", map[string]float64{})
		t.ctx.Outputs.SetVar("deltas", map[string]float64{})
	} else {
		if data, err := vars.GeneralizeData(out.passedAssertions); err == nil {
			t.ctx.Outputs.SetVar("passedAssertions", data)
		}

		if data, err := vars.GeneralizeData(out.failedAssertions); err == nil {
			t.ctx.Outputs.SetVar("failedAssertions", data)
		}

		if data, err := vars.GeneralizeData(out.values); err == nil {
			t.ctx.Outputs.SetVar("values", data)
		}

		if data, err := vars.GeneralizeData(out.deltas); err == nil {
			t.ctx.Outputs.SetVar("deltas", data)
		}
	}

	// Baselines
	if data, err := vars.GeneralizeData(t.baselines); err == nil {
		t.ctx.Outputs.SetVar("baselines", data)
	}

	t.ctx.Outputs.SetVar("scrapeErrors", t.scrapeErrors)
	t.ctx.Outputs.SetVar("assertionErrors", t.assertionErrors)
}

func countWaiting(results []assertionResult) int {
	count := 0

	for _, r := range results {
		if r.waiting {
			count++
		}
	}

	return count
}

func (t *Task) scrapeMetrics(ctx context.Context) (map[string]*dto.MetricFamily, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.config.URL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range t.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit response size
	maxSize := t.config.GetMaxResponseSizeBytes()
	limitedReader := io.LimitReader(resp.Body, maxSize+1)

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if int64(len(body)) > maxSize {
		return nil, fmt.Errorf("response body exceeds max size of %d bytes", maxSize)
	}

	// Parse Prometheus text format using UTF8 validation scheme
	parser := expfmt.NewTextParser(model.UTF8Validation)

	metricFamilies, err := parser.TextToMetricFamilies(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	return metricFamilies, nil
}

func (t *Task) evaluateAssertions(metricFamilies map[string]*dto.MetricFamily) []assertionResult {
	results := make([]assertionResult, 0, len(t.config.Assertions))

	for i := range t.config.Assertions {
		result := t.evaluateAssertion(&t.config.Assertions[i], metricFamilies)
		results = append(results, result)
	}

	return results
}

func (t *Task) evaluateAssertion(a *AssertionConfig, metricFamilies map[string]*dto.MetricFamily) assertionResult {
	result := assertionResult{
		name:  a.Name,
		value: math.NaN(),
		delta: math.NaN(),
	}

	// Get effective missing behaviors
	missingMetricBehavior := t.config.MissingMetric
	if a.MissingMetric != nil {
		missingMetricBehavior = *a.MissingMetric
	}

	missingSeriesBehavior := t.config.MissingSeries
	if a.MissingSeries != nil {
		missingSeriesBehavior = *a.MissingSeries
	}

	// Find the metric family
	mf, ok := metricFamilies[a.Metric]
	if !ok {
		return t.handleMissing(result, missingMetricBehavior, fmt.Sprintf("metric %q not found", a.Metric))
	}

	// Find matching series
	value, err := t.findMetricValue(mf, a.Labels)
	if err != nil {
		if strings.Contains(err.Error(), "no matching series") {
			return t.handleMissing(result, missingSeriesBehavior, err.Error())
		}

		result.err = err

		return result
	}

	result.value = value

	// Check for non-finite values
	if math.IsNaN(value) || math.IsInf(value, 0) {
		result.err = fmt.Errorf("metric value is non-finite: %v", value)

		return result
	}

	// Handle delta mode
	mode := a.Mode
	if mode == "" {
		mode = AssertionModeValue
	}

	var compareValue float64

	if mode == AssertionModeDelta {
		baseline, hasBaseline := t.baselines[a.Name]

		if !hasBaseline {
			// First scrape: record baseline, don't evaluate
			t.baselines[a.Name] = value
			result.waiting = true

			t.logger.Debugf("Assertion %q: recorded baseline %.4f", a.Name, value)

			return result
		}

		// Check for counter reset (only applies to COUNTER type, not gauges)
		// Gauges can legitimately decrease, so negative delta is normal for them
		isCounter := mf.GetType() == dto.MetricType_COUNTER
		if isCounter && value < baseline {
			switch t.config.ResetBehavior {
			case ResetBehaviorFail:
				result.err = fmt.Errorf("counter reset detected: current %.4f < baseline %.4f", value, baseline)

				return result
			case ResetBehaviorRebaseline:
				t.baselines[a.Name] = value
				t.logger.Warnf("Assertion %q: counter reset detected, rebaselining to %.4f", a.Name, value)

				result.waiting = true

				return result
			case ResetBehaviorIgnore:
				// Use old baseline, may produce negative delta
				t.logger.Warnf("Assertion %q: counter reset detected, ignoring", a.Name)
			}
		}

		compareValue = value - baseline
		result.delta = compareValue
	} else {
		compareValue = value
	}

	// Evaluate operator
	result.passed = evaluateOperator(a.Operator, compareValue, a.Value)

	if !result.passed {
		t.logger.Debugf("Assertion %q failed: %.4f %s %.4f = false", a.Name, compareValue, a.Operator, a.Value)
	}

	return result
}

func (t *Task) handleMissing(result assertionResult, behavior MissingBehavior, message string) assertionResult {
	switch behavior {
	case MissingBehaviorFail:
		result.err = errors.New(message)
	case MissingBehaviorPass:
		result.passed = true
	case MissingBehaviorWait:
		result.waiting = true
	default:
		result.waiting = true
	}

	return result
}

func (t *Task) findMetricValue(mf *dto.MetricFamily, labels map[string]string) (float64, error) {
	var matchingMetrics []*dto.Metric

	for _, m := range mf.GetMetric() {
		if matchLabels(m.GetLabel(), labels) {
			matchingMetrics = append(matchingMetrics, m)
		}
	}

	if len(matchingMetrics) == 0 {
		return 0, fmt.Errorf("no matching series for labels %v", labels)
	}

	if len(matchingMetrics) > 1 {
		return 0, fmt.Errorf("labels match %d series (must match exactly one)", len(matchingMetrics))
	}

	value, err := getMetricValue(matchingMetrics[0], mf.GetType())
	if err != nil {
		return 0, fmt.Errorf("failed to extract metric value: %w", err)
	}

	return value, nil
}

func matchLabels(metricLabels []*dto.LabelPair, wantLabels map[string]string) bool {
	if len(wantLabels) == 0 {
		return true
	}

	labelMap := make(map[string]string, len(metricLabels))

	for _, lp := range metricLabels {
		labelMap[lp.GetName()] = lp.GetValue()
	}

	for k, v := range wantLabels {
		if labelMap[k] != v {
			return false
		}
	}

	return true
}

var errNoMetricValue = errors.New("metric has no extractable value")

func getMetricValue(m *dto.Metric, mType dto.MetricType) (float64, error) {
	switch mType {
	case dto.MetricType_COUNTER:
		if c := m.GetCounter(); c != nil {
			return c.GetValue(), nil
		}
	case dto.MetricType_GAUGE:
		if g := m.GetGauge(); g != nil {
			return g.GetValue(), nil
		}
	case dto.MetricType_UNTYPED:
		if u := m.GetUntyped(); u != nil {
			return u.GetValue(), nil
		}
	case dto.MetricType_SUMMARY:
		if s := m.GetSummary(); s != nil {
			return s.GetSampleSum(), nil
		}
	case dto.MetricType_HISTOGRAM, dto.MetricType_GAUGE_HISTOGRAM:
		if h := m.GetHistogram(); h != nil {
			return h.GetSampleSum(), nil
		}
	}

	// Fallback: try all types
	if c := m.GetCounter(); c != nil {
		return c.GetValue(), nil
	}

	if g := m.GetGauge(); g != nil {
		return g.GetValue(), nil
	}

	if u := m.GetUntyped(); u != nil {
		return u.GetValue(), nil
	}

	return 0, errNoMetricValue
}

func evaluateOperator(op Operator, actual, expected float64) bool {
	switch op {
	case OperatorEq:
		return actual == expected
	case OperatorNeq:
		return actual != expected
	case OperatorGt:
		return actual > expected
	case OperatorGte:
		return actual >= expected
	case OperatorLt:
		return actual < expected
	case OperatorLte:
		return actual <= expected
	default:
		return false
	}
}
