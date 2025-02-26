## `tx_pool_check` Task

### Description

The `tx_pool_check` task evaluates the throughput and latency of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`txCount`**:
  The total number of transactions to send.

- **`measureInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

- **`expectedLatency`**:
  The expected average transaction latency in milliseconds.

- **`failOnHighLatency`**:
  Whether the task should fail if the measured latency exceeds `expectedLatency`.

- **`clientPattern`**:
  Regex pattern to select specific client endpoints.

- **`excludeClientPattern`**:
  Regex pattern to exclude certain clients.

### Outputs

- **`avgLatency`**:
  The measured average transaction latency in milliseconds.

- **`totalTime`**:
  The total time taken to send all transactions.

### Defaults

```yaml
- name: tx_pool_check
  config:
    txCount: 1000
    measureInterval: 100
    expectedLatency: 500
    failOnHighLatency: false
    clientPattern: ""
    excludeClientPattern: ""
```
