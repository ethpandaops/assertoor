# `check_consensus_identity` Task

This task checks consensus client node identity information by querying the `/eth/v1/node/identity` API endpoint. It can verify various aspects of the node identity including CGC (Custody Group Count) extracted from ENR (Ethereum Node Record).

## Task Behavior

- The task polls clients at regular intervals to check their identity information.
- By default, the task returns immediately when the identity criteria are met.
- Use `continueOnPass: true` to keep monitoring even after success.

## Configuration

### Required Parameters
- **`clientPattern`** *(string)*: Pattern to match client names (e.g., `"lodestar-*"`, `"*"` for all)

### Optional Parameters
- **`pollInterval`** *(duration)*: Interval between checks (default: `10s`)
- **`minClientCount`** *(int)*: Minimum number of clients that must pass checks (default: `1`)
- **`maxFailCount`** *(int)*: Maximum number of clients that can fail (-1 for no limit, default: `-1`)
- **`failOnCheckMiss`** *(bool)*: Whether to fail the task when checks don't pass (default: `false`)
- **`continueOnPass`** *(bool)*: Keep monitoring even after success (default: `false`)

### CGC (Custody Group Count) Checks
- **`expectCgc`** *(int)*: Expect exact CGC value
- **`minCgc`** *(int)*: Minimum CGC value required
- **`maxCgc`** *(int)*: Maximum CGC value allowed

### ENR Checks
- **`expectEnrField`** *(map[string]interface{})*: Expected ENR field values

### PeerID Checks
- **`expectPeerIdPattern`** *(string)*: Regex pattern that PeerID must match

### P2P Address Checks
- **`expectP2pAddressCount`** *(int)*: Expected number of P2P addresses
- **`expectP2pAddressMatch`** *(string)*: Regex pattern that at least one P2P address must match

### Metadata Checks
- **`expectSeqNumber`** *(uint64)*: Expected sequence number
- **`minSeqNumber`** *(uint64)*: Minimum sequence number required

## Outputs

The task exports the following data via `ctx.Outputs`:

- **`matchingClients`**: Array of clients that passed all checks
- **`failedClients`**: Array of clients that failed checks
- **`totalCount`**: Total number of clients checked
- **`matchingCount`**: Number of clients that passed checks
- **`failedCount`**: Number of clients that failed checks

Each client result includes:
- `clientName`: Name of the consensus client
- `peerId`: Peer ID from node identity
- `enr`: ENR string
- `p2pAddresses`: Array of P2P addresses
- `discoveryAddresses`: Array of discovery addresses
- `seqNumber`: Metadata sequence number
- `attnets`: Attestation subnets
- `syncnets`: Sync subnets
- `cgc`: Extracted Custody Group Count
- `enrFields`: Parsed ENR fields
- `checksPassed`: Whether all configured checks passed
- `failureReasons`: Array of reasons why checks failed (if any)

## Example Configurations

### Basic Identity Check
```yaml
- name: check_node_identity
  task: check_consensus_identity
  config:
    clientPattern: "lodestar-*"
    minClientCount: 1
```

### CGC Validation
```yaml
- name: validate_cgc
  task: check_consensus_identity
  config:
    clientPattern: "*"
    expectCgc: 8
    failOnCheckMiss: true
```

### Continuous Monitoring
```yaml
- name: monitor_identity
  task: check_consensus_identity
  config:
    clientPattern: "*"
    minCgc: 4
    continueOnPass: true
    timeout: 30m
```

### Comprehensive Identity Check
```yaml
- name: full_identity_check
  task: check_consensus_identity
  config:
    clientPattern: "prysm-*"
    minCgc: 4
    maxCgc: 16
    expectP2pAddressCount: 2
    expectPeerIdPattern: "^16Uiu2HA.*"
    minSeqNumber: 1
    pollInterval: 30s
    failOnCheckMiss: true
```

## Use Cases

1. **PeerDAS Validation**: Verify nodes have correct custody assignments
2. **Network Health**: Check node identity consistency across clients
3. **Configuration Validation**: Ensure nodes are properly configured for specific network requirements
4. **Testing**: Validate node behavior changes after deposits or configuration updates
5. **Monitoring**: Track node identity changes over time
