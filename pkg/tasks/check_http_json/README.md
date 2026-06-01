## `check_http_json` Task

### Description
The `check_http_json` task fetches JSON from an HTTP endpoint and evaluates assertions using jq queries.

#### Task Behavior
- The task polls the endpoint at regular intervals.
- By default, the task returns immediately when all assertions pass.
- Use `continueOnPass: true` to keep monitoring even after success.
- Use `failOnCheckMiss: true` to fail immediately when assertions are not met.
- Empty assertions allow status-only checks (verify HTTP status without parsing body).

### Configuration Parameters

- **`url`**:\
  HTTP URL of the JSON endpoint. Required.

- **`method`**:\
  HTTP method: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, or `HEAD`. Default: `GET`.

- **`headers`**:\
  Optional HTTP request headers (e.g., for authentication). Default: `{}`.

- **`body`**:\
  Request body as YAML/JSON value. JSON-encoded before sending.

- **`bodyRaw`**:\
  Raw request body sent as-is. Takes precedence over `body`.

- **`expectStatus`**:\
  Expected HTTP status code. Default: `200`.

- **`expectStatuses`**:\
  Multiple expected HTTP status codes. Cannot be used with `expectStatus`.

- **`pollInterval`**:\
  Interval between requests. Default: `10s`.

- **`requestTimeout`**:\
  Timeout for a single HTTP request. Default: `5s`.

- **`maxResponseSize`**:\
  Maximum response body size. Must be positive. Default: `10MB`.

- **`failOnCheckMiss`**:\
  If `true`, fail immediately when assertions are not met. If `false`, keep polling until timeout or success. Default: `false`.

- **`continueOnPass`**:\
  If `true`, continue checking after all assertions pass. Default: `false`.

- **`assertions`**:\
  List of JSON assertions. Empty list allowed for status-only checks.

#### Assertion Configuration

- **`name`**: Unique assertion name. Required.
- **`query`**: jq expression evaluated against the JSON response. Required.
- **`exists`**: Assert whether the query returns at least one non-null result. Use for existence checks.
- **`operator`**: Comparison operator: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `contains`, `not_contains`.
- **`value`**: Expected value for comparison.
- **`allowMissing`**: If `true`, missing results are treated as pass for this assertion.

Each assertion must use exactly one mode: `exists` OR `operator`+`value`.

#### Assertion Modes

**Existence Mode (`exists`)**:
- `exists: true`: Pass if query returns at least one non-null result.
- `exists: false`: Pass if query returns no results or only null.

**Comparison Mode (`operator`)**:
- Query must return exactly one result for scalar comparisons.
- Supports type-safe comparisons (numeric, string, bool, array, object).
- `contains` checks substring, array element, or object key/value subset.

#### Examples

```yaml
assertions:
  # Check boolean field is true
  - name: service_ready
    query: '.ready'
    operator: eq
    value: true

  # Check field exists
  - name: has_items
    query: '.items'
    exists: true

  # Check nested value
  - name: nested_check
    query: '.data.config.enabled'
    operator: eq
    value: true

  # Check array length
  - name: enough_items
    query: '.items | length'
    operator: gte
    value: 3

  # Check array contains element (using jq select)
  - name: has_reth_verifier
    query: '.proof_types[] | select(.proof_type == "reth-zisk" and .can_verify == true)'
    exists: true

  # String contains substring
  - name: status_contains_ok
    query: '.status'
    operator: contains
    value: "OK"
```

#### Status-Only Checks

For endpoints that return non-JSON responses or when only the HTTP status matters:

```yaml
- name: check_http_json
  config:
    url: http://service/health
    expectStatus: 200
    assertions: []
```

#### HEAD Requests

HEAD requests are status-only and cannot have assertions:

```yaml
- name: check_http_json
  config:
    url: http://service/health
    method: HEAD
    expectStatus: 200
    assertions: []
```

### Important Notes

#### Null Value Handling

To check if a field is null or missing, use the existence mode rather than comparison:

```yaml
# Check if field exists and is non-null
- name: has_data
  query: '.data'
  exists: true

# Check if field is null or missing
- name: no_data
  query: '.data'
  exists: false
```

Using `operator: eq` with `value: null` is not supported because the YAML `null` value is converted to Go's `nil`, which is treated as a missing required field. Use `exists: false` instead.

#### Request Body

The `body: null` default means no request body is sent. To explicitly send a JSON `null` value as the body, use `bodyRaw: "null"`.

#### Numeric Precision

Numeric comparisons convert all integer types to float64 for type-agnostic comparison (JSON numbers decode as float64, while YAML integers decode as int). This works correctly for values up to 2^53. For very large integers (IDs, counters, timestamps beyond 2^53), precision loss may cause incorrect comparisons. Use string comparison for such values.

### Outputs

- **`passedAssertions`**: Array of assertion names that passed.
- **`failedAssertions`**: Array of assertion names that failed.
- **`values`**: Map of assertion name to latest jq result.
- **`httpStatus`**: Latest HTTP status code.
- **`responseErrors`**: Number of HTTP/read/status errors.
- **`parseErrors`**: Number of JSON parse errors.
- **`assertionErrors`**: Number of assertion evaluation errors.

### Defaults

```yaml
- name: check_http_json
  config:
    url: ""
    method: GET
    headers: {}
    body: null
    bodyRaw: ""
    expectStatus: 200
    pollInterval: 10s
    requestTimeout: 5s
    maxResponseSize: 10MB
    failOnCheckMiss: false
    continueOnPass: false
    assertions: []
```
