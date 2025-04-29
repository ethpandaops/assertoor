## `tx_pool_latency_analysis` Task

### Description

The `tx_pool_latency_analysis` task evaluates latency of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`privateKey`**:
  The private key of the account to use for sending transactions.

- **`txCount`**:
  The total number of transactions to send.

- **`measureInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

- **`highLatency`**:
  The expected average transaction latency in milliseconds.

- **`failOnHighLatency`**:
  Whether the task should fail if the measured latency exceeds `highLatency`.


### Outputs

- **`tx_count`**:
  The total number of transactions sent.

- **`avg_latency_ms`**:
  The average latency of the transactions in milliseconds.

### Defaults

```yaml
- name: tx_pool_latency_analysis
  config:
    txCount: 15000
    measureInterval: 1000
    highLatency: 5000
    failOnHighLatency: false
  configVars:
    privateKey: "tx_pool_latency_analysis"
```
