## `tx_pool_throughput_analysis` Task

### Description

The `tx_pool_throughput_analysis` task evaluates the throughput of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`privateKey`**:
  The private key of the account to use for sending transactions.

- **`qps`**:
  The total number of transactions to send in one second.

- **`measureInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

### Outputs

- **`total_time_mus`**:
  The total time taken to send the transactions in microseconds.

### Defaults

```yaml
- name: tx_pool_throughput_analysis
  config:
    qps: 15000
    measureInterval: 1000
  configVars:
    privateKey: "walletPrivkey"
```
