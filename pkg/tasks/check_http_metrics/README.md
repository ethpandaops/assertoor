## `check_http_metrics` Task

### Description
The `check_http_metrics` task fetches metrics from an HTTP Prometheus endpoint and evaluates assertions against metric values.

#### Task Behavior
- The task polls the metrics endpoint at regular intervals.
- By default, the task returns immediately when all assertions pass.
- Use `continueOnPass: true` to keep monitoring even after success.
- Use `failOnCheckMiss: true` to fail immediately when assertions are not met.

### Configuration Parameters

- **`url`**:\
  HTTP URL of the Prometheus metrics endpoint. Required.

- **`headers`**:\
  Optional HTTP request headers (e.g., for authentication). Default: `{}`.

- **`pollInterval`**:\
  Interval between metric scrapes. Default: `10s`.

- **`requestTimeout`**:\
  Timeout for a single HTTP request. Default: `5s`.

- **`maxResponseSize`**:\
  Maximum response body size. Must be positive. Default: `10MB`.

- **`failOnCheckMiss`**:\
  If `true`, fail immediately when assertions are not met. If `false`, keep polling until timeout or success. Default: `false`.

- **`continueOnPass`**:\
  If `true`, continue checking after all assertions pass. Default: `false`.

- **`missingMetric`**:\
  Behavior when a metric family is missing: `wait`, `fail`, or `pass`. Default: `wait`.

- **`missingSeries`**:\
  Behavior when no time series matches the label selector: `wait`, `fail`, or `pass`. Default: `wait`.

- **`resetBehavior`**:\
  Behavior when a COUNTER metric's value drops below baseline (indicating restart): `fail`, `rebaseline`, or `ignore`. Only applies to COUNTER type metrics. Default: `fail`.

- **`assertions`**:\
  List of metric assertions. At least one required.

#### Assertion Configuration

- **`name`**: Unique assertion name. Required.
- **`metric`**: Prometheus metric name. Required.
- **`labels`**: Label selector (subset matching, exact values).
- **`labelPatterns`**: Label selector with regex values, each matching the whole label value (e.g. `status: "[45].."`). Combined with `labels`.
- **`aggregate`**: How matching series are combined. `single` (default): the selector must match exactly one series. `sum`: the sum of all matching series is compared. `each`: every matching series must pass on its own, failures name the offending series.
- **`mode`**: `value` (current value) or `delta` (change since baseline). Default: `value`.
- **`operator`**: Comparison operator: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`. Required.
- **`value`**: Expected numeric value. Required.
- **`missingMetric`**: Per-assertion override for global `missingMetric`.
- **`missingSeries`**: Per-assertion override for global `missingSeries`.

#### Delta Mode

In `delta` mode, the task tracks changes over time:
1. First scrape: records the current value as baseline (waits, does not evaluate)
2. Subsequent scrapes: computes `delta = current - baseline` and evaluates

Negative deltas are valid for GAUGE and UNTYPED metrics. For COUNTER metrics, a decrease triggers `resetBehavior`.

With `aggregate: sum` or `each`, baselines are kept per series. The first scrape records every matching series (even if there are none yet), and a series that first appears later counts from zero, so an error counter created during the run is still caught.

#### Examples

```yaml
assertions:
  # Fail on any new 4xx/5xx REST response, naming each route and status that occurred
  - name: no_rest_errors
    metric: lodestar_api_rest_errors_total
    aggregate: each
    mode: delta
    operator: eq
    value: 0
    missingMetric: pass
    missingSeries: pass

  # Sum all error responses across routes
  - name: few_errors
    metric: ngx_request_count
    labelPatterns:
      status: "[45].."
    aggregate: sum
    mode: delta
    operator: lte
    value: 5

  # Check counter increased by at least 1
  - name: counter_increased
    metric: my_counter
    labels:
      env: prod
    mode: delta
    operator: gte
    value: 1

  # Check gauge decreased (negative delta)
  - name: gauge_dropped
    metric: my_gauge
    mode: delta
    operator: lte
    value: -1

  # Check current value is above threshold
  - name: value_above_threshold
    metric: my_metric
    operator: gt
    value: 100
```

#### Metric Type Handling

| Type | Value Extracted |
|------|-----------------|
| COUNTER | Counter value |
| GAUGE | Gauge value |
| UNTYPED | Untyped value |
| SUMMARY | Sample sum |
| HISTOGRAM | Sample sum |

Counter reset detection only applies to COUNTER type. SUMMARY and HISTOGRAM use sample sum; bucket/quantile helpers are not supported.

### Outputs

- **`passedAssertions`**: Array of assertion names that passed.
- **`failedAssertions`**: Array of assertion names that failed.
- **`values`**: Map of assertion name to latest observed value.
- **`deltas`**: Map of assertion name to computed delta (for `delta` mode).
- **`baselines`**: Map of assertion name to baseline value (for `delta` mode).
- **`scrapeErrors`**: Number of HTTP/parsing errors.
- **`assertionErrors`**: Number of assertion evaluation errors.

### Defaults

```yaml
- name: check_http_metrics
  config:
    url: ""
    headers: {}
    pollInterval: 10s
    requestTimeout: 5s
    maxResponseSize: 10MB
    failOnCheckMiss: false
    continueOnPass: false
    missingMetric: wait
    missingSeries: wait
    resetBehavior: fail
    assertions: []
```
