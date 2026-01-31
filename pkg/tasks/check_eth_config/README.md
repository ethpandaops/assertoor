## `check_eth_config` Task

### Description
The `check_eth_config` task verifies that all execution clients in the network return consistent chain configuration via the `eth_config` JSON-RPC method as defined in EIP-7910. This task is essential for ensuring that all execution layer clients have the same fork configuration, including chain ID, fork IDs, activation times, precompiles, and system contracts. When mismatches are detected, the task provides a detailed diff showing which clients returned which configuration variants.

### Configuration Parameters

- **`clientPattern`**:
  A regex pattern to select specific execution client endpoints for querying `eth_config`. This allows targeting specific clients within the network. An empty pattern (default) targets all ready execution clients.

- **`excludeClientPattern`**:
  A regex pattern to exclude certain execution clients from the `eth_config` check. This is useful for excluding known misconfigured or test clients from the consistency check.

- **`failOnMismatch`**:
  Determines whether the task should fail if any execution client returns a different `eth_config` response. If set to `true` (default), the task fails on configuration mismatches. If set to `false`, mismatches are logged but the task continues without failure.

- **`excludeSyncingClients`**:
  When set to `true`, the task excludes execution clients that are currently syncing. If set to `false` (default), syncing clients are included in the check. This is useful for testing configuration consistency even before clients are fully synced, as `eth_config` returns configuration data that doesn't depend on sync status.

### Outputs

- **`ethConfig`**:
  The reference `eth_config` response from the first successful client query, returned as a JSON string. This output contains the complete fork configuration including current, next, and last fork details with activation times, chain ID, fork ID, blob schedule, precompiles, and system contracts.

### Defaults

Default settings for the `check_eth_config` task:

```yaml
- name: check_eth_config
  config:
    clientPattern: ""
    excludeClientPattern: ""
    failOnMismatch: true
    excludeSyncingClients: false
```

### Example Usage

Basic usage checking all execution clients:

```yaml
- name: check_eth_config
  title: "Verify eth_config consistency across all EL clients"
  config:
    failOnMismatch: true
```

Checking specific clients only:

```yaml
- name: check_eth_config
  title: "Verify eth_config for Geth clients only"
  config:
    clientPattern: ".*geth.*"
    failOnMismatch: true
```

Non-blocking check that logs mismatches but doesn't fail:

```yaml
- name: check_eth_config
  title: "Monitor eth_config consistency"
  config:
    failOnMismatch: false
```

Only check fully synced clients:

```yaml
- name: check_eth_config
  title: "Verify eth_config for synced clients only"
  config:
    excludeSyncingClients: true
    failOnMismatch: true
```

### Implementation Details

The task queries the `eth_config` RPC method (EIP-7910) from all matching execution clients and performs a JSON-level comparison of the responses. When configurations match, the task succeeds and outputs the reference configuration. When mismatches are detected, the task provides a detailed error log showing:

- All unique configuration variants encountered
- Which clients returned each variant
- The full JSON structure of each variant for easy comparison

This makes it easy to diagnose configuration drift or misconfiguration across the execution layer.
