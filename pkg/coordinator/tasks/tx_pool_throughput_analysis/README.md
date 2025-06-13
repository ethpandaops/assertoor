## `tx_pool_throughput_analysis` Task

### Description

The `tx_pool_throughput_analysis` task evaluates the throughput of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`privateKey`**:
  The private key of the account to use for sending transactions.

- **`tps`**:
  The total number of transactions to send in one second.

- **`duration_s`**:
  The test duration (the number of transactions to send is calculated as `tps * duration_s`).

- **`logInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

### Outputs

- **`tx_count`**:
  The total number of transactions sent.

- **`mean_tps_throughput`**:
  The mean throughput (tps)

### Defaults

```yaml
- name: tx_pool_throughput_analysis
  config:
    tps: 1000
    duration_s: 10  
    logInterval: 1000
  configVars:
    privateKey: "walletPrivkey"
```
