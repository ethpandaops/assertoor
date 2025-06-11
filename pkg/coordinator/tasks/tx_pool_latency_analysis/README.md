## `tx_pool_latency_analysis` Task

### Description

The `tx_pool_latency_analysis` task evaluates latency of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`privateKey`**:
  The private key of the account to use for sending transactions.

- **`tps`**:
  The total number of transactions to send in one second.

- **`duration_s`**:
  The test duration (the number of transactions to send is calculated as `tps * duration_s`).

- **`measureInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

### Outputs

- **`tx_count`**:
  The total number of transactions sent.

- **`max_latency_ms`**:
  The average latency of the transactions in milliseconds.

### Defaults

```yaml
- name: tx_pool_latency_analysis
  config:
    tps: 100
    duration_s: 10  
    measureInterval: 1000
  configVars:
    privateKey: "walletPrivkey"
```


