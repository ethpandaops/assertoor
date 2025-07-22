## `tx_pool_latency_analysis` Task

### Description

The `tx_pool_latency_analysis` task evaluates latency of transaction processing within an Ethereum execution client’s transaction pool.

### Configuration Parameters

- **`privateKey`**:
  The private key of the account to use for sending transactions.

- **`tps`**:
  The total number of transactions to send in one second.

- **`durationS`**:
  The test duration (the number of transactions to send is calculated as `tps * durationS`).

- **`logInterval`**:
  The interval at which the script logs progress (e.g., every 100 transactions).

### Outputs

- **`tx_count`**:
  The total number of transactions sent.

- **`min_latency_mus`**:
  The min latency of the transactions in microseconds.

- **`max_latency_mus`**:
  The max latency of the transactions in microseconds.

- **`tx_pool_latency_hdr_plot`**:
  The HDR plot of the transaction pool latency.

- **`duplicated_p2p_event_count`**:
  The number of duplicated P2P events.

- **`missed_p2p_event_count`**:
  The number of missed P2P events.

- **`coordinated_omission_event_count`**:
  The number of coordinated omission events.

- **`duplicated_p2p_event_count_percentage`**:
  The percentage of duplicated P2P events.

- **`missed_p2p_event_count_percentage`**:
  The percentage of missed P2P events.

- **`coordinated_omission_event_count_percentage`**:
  The percentage of coordinated omission events.

### Defaults

```yaml
- name: tx_pool_latency_analysis
  config:
    tps: 1000
    durationS: 10  
    logInterval: 1000
  configVars:
    privateKey: "walletPrivkey"
```
