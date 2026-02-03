# `get_consensus_validators` Task

This task retrieves validators from the consensus layer that match specified filtering criteria. It's useful for finding validators associated with specific clients, status conditions, or other attributes.

## Configuration

### Client/Name Filtering
- **`clientPattern`** *(string)*: Regex pattern to match client names in validator names
- **`validatorNamePattern`** *(string)*: Regex pattern to match validator names directly

### Status Filtering
- **`validatorStatus`** *([]string)*: Array of allowed validator statuses (default: all statuses)
  - Possible values: `pending_initialized`, `pending_queued`, `active_ongoing`, `active_exiting`, `active_slashed`, `exited_unslashed`, `exited_slashed`, `withdrawal_possible`, `withdrawal_done`

### Balance Filtering
- **`minValidatorBalance`** *(uint64)*: Minimum validator balance in Gwei
- **`maxValidatorBalance`** *(uint64)*: Maximum validator balance in Gwei

### Index Filtering
- **`minValidatorIndex`** *(uint64)*: Minimum validator index
- **`maxValidatorIndex`** *(uint64)*: Maximum validator index

### Other Filtering
- **`withdrawalCredsPrefix`** *(string)*: Required prefix for withdrawal credentials (e.g., "0x01", "0x02")

### Output Options
- **`maxResults`** *(int)*: Maximum number of validators to return (default: 100)
- **`outputFormat`** *(string)*: Output format - "full", "pubkeys", or "indices" (default: "full")

## Outputs

Depending on `outputFormat`, the task exports:

### Format: "full" (default)
- **`validators`**: Array of full validator information objects
- **`count`**: Number of matching validators

Each validator object includes:
- `index`: Validator index
- `pubkey`: Validator public key (0x prefixed hex)
- `balance`: Current balance in Gwei
- `status`: Validator status string
- `effectiveBalance`: Effective balance in Gwei
- `withdrawalCredentials`: Withdrawal credentials (0x prefixed hex)
- `activationEpoch`: Activation epoch
- `exitEpoch`: Exit epoch
- `withdrawableEpoch`: Withdrawable epoch
- `slashed`: Boolean indicating if validator is slashed

### Format: "pubkeys"
- **`pubkeys`**: Array of validator public keys as hex strings
- **`count`**: Number of matching validators

### Format: "indices"
- **`indices`**: Array of validator indices as integers
- **`count`**: Number of matching validators

## Example Configurations

### Find Validators for Specific Client
```yaml
- name: find_lighthouse_validators
  task: get_consensus_validators
  config:
    clientPattern: "lighthouse.*"
    validatorStatus: ["active_ongoing"]
    maxResults: 10
    outputFormat: "full"
```

### Get Validator Pubkeys for BLS Changes
```yaml
- name: get_validator_pubkeys
  task: get_consensus_validators
  config:
    validatorNamePattern: "validator_[0-9]+"
    validatorStatus: ["active_ongoing"]
    withdrawalCredsPrefix: "0x01"
    outputFormat: "pubkeys"
    maxResults: 5
```

### Find Validators by Balance Range
```yaml
- name: find_rich_validators
  task: get_consensus_validators
  config:
    clientPattern: "prysm.*"
    minValidatorBalance: 40000000000  # > 40 ETH
    validatorStatus: ["active_ongoing"]
    maxResults: 20
```

### Get Validator Indices for Operations
```yaml
- name: get_validator_indices
  task: get_consensus_validators
  config:
    validatorNamePattern: "test_validator_.*"
    validatorStatus: ["pending_initialized", "pending_queued"]
    outputFormat: "indices"
    maxResults: 2
```

## Use Cases

1. **Client-Specific Operations**: Find validators belonging to specific consensus clients
2. **BLS Changes**: Identify validators that need withdrawal credential updates
3. **Deposit Operations**: Find validators for top-up deposits
4. **Status Monitoring**: Track validators in specific states
5. **Balance Analysis**: Identify validators by balance criteria
6. **Bulk Operations**: Get validator sets for matrix operations