# `check_consensus_cgc` Task

This task checks the CGC (Custody Group Count) value in consensus layer client ENR records.

## Description

The CGC field in ENR records indicates the custody responsibilities of a consensus layer node:
- **0x04**: Default value for non-validating consensus layer nodes
- **0x08**: Default value for validating consensus layer nodes
- **0x08 + N**: For validating nodes, where N represents the number of additional 32 ETH chunks being custodied

## Configuration Parameters

- **`clientPattern`** *(string)*: Pattern to match client names (default: empty, matches all)
- **`pollInterval`** *(duration)*: How often to check CGC values (default: 30s)
- **`expectedCgcValue`** *(int)*: Specific CGC value to expect (overrides other checks if set)
- **`expectedNonValidating`** *(int)*: Expected CGC value for non-validating nodes (default: 0x04)
- **`expectedValidating`** *(int)*: Expected CGC value for validating nodes (default: 0x08)
- **`minClientCount`** *(int)*: Minimum number of clients that must pass (default: all)
- **`failOnCheckMiss`** *(bool)*: Whether to fail the task if checks don't pass (default: false)
- **`resultVar`** *(string)*: Variable name to store the first valid CGC value found

## Outputs

The task provides these output variables:

- **`validClients`**: Array of clients with valid CGC values
- **`invalidClients`**: Array of clients with invalid CGC values
- **`totalCount`**: Total number of clients checked
- **`validCount`**: Number of clients with valid CGC values
- **`invalidCount`**: Number of clients with invalid CGC values

## Example Usage

### Basic CGC Check
```yaml
- name: check_consensus_cgc
  title: "Check CGC values in ENR records"
  config:
    clientPattern: "beacon-*"
    pollInterval: 30s
    failOnCheckMiss: true
```

### Check for Specific CGC Value
```yaml
- name: check_consensus_cgc
  title: "Verify specific CGC value"
  config:
    expectedCgcValue: 12  # Expecting 0x08 + 4 (validating with 4 additional 32 ETH chunks)
    minClientCount: 1
    resultVar: "detected_cgc_value"
```

### Check Non-Validating Nodes
```yaml
- name: check_consensus_cgc
  title: "Verify non-validating nodes"
  config:
    expectedCgcValue: 4  # 0x04 for non-validating
    clientPattern: "non-validator-*"
```

## Notes

- The task extracts ENR records from the `/eth/v1/node/identity` beacon API endpoint
- CGC parsing is currently simplified and may need enhancement for production use
- If no CGC field is found in the ENR, the task assumes the default non-validating value (0x04)
- The task validates that CGC values follow the expected pattern for validating vs non-validating nodes