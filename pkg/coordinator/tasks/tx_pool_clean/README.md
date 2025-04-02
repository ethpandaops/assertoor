## `tx_pool_clean` Task

### Description

The `tx_pool_clean` task is designed to monitor the transaction pool of a blockchain node. It checks for the presence of transactions in the pool and wait fors for a specified time before re-checking. This task is useful for ensuring that transactions are being processed and not stuck in the pool.

<!-- ### Configuration Parameters

- **`waitTime`**:
  The time to wait in seconds before re-checking the transaction pool for a client. Default is `5`. -->

### Defaults

```yaml
- name: tx_pool_clean
  config:
    waitTime: 5
```
