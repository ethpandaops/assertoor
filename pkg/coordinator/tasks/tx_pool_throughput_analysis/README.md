## `tx_pool_throughput_analysis` Task

### Description

The `tx_pool_throughput_analysis` task evaluates the throughput of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`privateKey`**:
  The private key of the account to use for sending transactions.

- **`txCount`**:
  The total number of transactions to send.

- **`measureInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

- **`secondsBeforeRunning`**:
  The number of seconds to wait before starting the transaction sending process.

### Outputs

- **`total_time_ms`**:
  The total time taken to send the transactions in milliseconds.

### Defaults

```yaml
- name: tx_pool_throughput_analysis
  config:
    nonce: 0
    txCount: 15000
    measureInterval: 1000
  configVars:
    privateKey: "walletPrivkey"
```
