## `tx_pool_throughput_analysis` Task

### Description

The `tx_pool_throughput_analysis` task evaluates the throughput of transaction processing within an Ethereum execution client’s transaction pool.

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

- **`throughput_measures`**:
  An array of throughput measurement objects, each containing:
  - `load_tps`: The sending TPS for this measurement
  - `processed_tps`: The actual processed TPS achieved
  - `not_received_p2p_event_count`: Count of transactions that didn't receive P2P events
  - `coordinated_omission_event_count`: Count of coordinated omission events

- **`total_sent_tx`**:
  The total number of transactions sent across all TPS measurements.

- **`missed_p2p_event_count`**:
  The total count of missed P2P events across all measurements.

- **`coordinated_omission_event_count`**:
  The total count of coordinated omission events across all measurements.

- **`missed_p2p_event_count_percentage`**:
  The percentage of transactions that missed P2P events.

- **`coordinated_omission_event_count_percentage`**:
  The percentage of transactions with coordinated omission events.

- **`starting_tps`**:
  The starting TPS value used in the test.

- **`ending_tps`**:
  The ending TPS value used in the test.

- **`increment_tps`**:
  The TPS increment value used between measurements.

- **`duration_s`**:
  The duration in seconds for each TPS measurement.

### Defaults

```yaml
- name: tx_pool_throughput_analysis
  config:
    tps: 1000
    durationS: 10  
    logInterval: 1000
  configVars:
    privateKey: "walletPrivkey"
```
