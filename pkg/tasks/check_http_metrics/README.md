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
  List of metric assertions. At least one required. Example:
  ```yaml
  - { "name": "counter_increased", "metric": "my_counter", "labels": { "env": "prod" }, "mode": "delta", "operator": "gt", "value": 0 }
  ```

#### Assertion Configuration

- **`name`**: Unique assertion name. Required.
- **`metric`**: Prometheus metric name. Required.
- **`labels`**: Label selector (subset matching). Must match exactly one series.
- **`mode`**: `value` (current value) or `delta` (change since baseline). Default: `value`.
- **`operator`**: Comparison operator: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`. Required.
- **`value`**: Expected numeric value. Required.
- **`missingMetric`**: Per-assertion override for global `missingMetric`.
- **`missingSeries`**: Per-assertion override for global `missingSeries`.

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
