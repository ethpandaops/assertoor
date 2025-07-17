# Check Consensus Config Spec Task

This task validates that consensus clients properly implement the `/eth/v1/config/spec` endpoint according to the Ethereum consensus specification.

## Description

The task fetches the latest consensus specification from the Ethereum consensus-specs repository, combining the main config with all relevant preset files (phase0, altair, bellatrix, capella, deneb, electra, fulu, etc.), and compares it against the response from each consensus client's `/eth/v1/config/spec` endpoint. It checks for:

1. **Missing required fields** - Fields that should be present according to the spec but are missing from the client response
2. **Extra fields** - Fields in the client response that are not in the specification (can be allowed or disallowed)
3. **Value mismatches** - Fields where the client returns a different value than expected

## Configuration

```yaml
- name: check_consensus_config_spec
  title: "Validate consensus config spec endpoint"
  config:
    clientPattern: ".*"  # Regex pattern to filter clients
    pollInterval: 10s    # How often to check clients
    networkPreset: "mainnet"  # Network preset (mainnet or minimal only)
    specBranch: "dev"    # Git branch to use (dev, master, etc.)
    presetFiles: []      # Optional: override default preset files
    requiredFields: []   # If empty, uses all fields from combined spec
    allowExtraFields: true  # Whether to allow extra fields not in spec
```

## Parameters

- `clientPattern`: Regex pattern to filter which clients to check (default: ".*" - all clients)
- `pollInterval`: Interval between checks (default: 10s)
- `networkPreset`: Network preset to use (default: "mainnet", can be "mainnet" or "minimal" only)
- `specBranch`: Git branch to use for fetching specs (default: "dev")
- `presetFiles`: List of preset files to fetch and combine (default: ["phase0.yaml", "altair.yaml", "bellatrix.yaml", "capella.yaml", "deneb.yaml", "electra.yaml", "fulu.yaml"])
- `requiredFields`: List of fields that must be present (default: empty, uses all fields from combined spec)
- `allowExtraFields`: Whether to allow fields not in the specification (default: true)

**Note**: The task automatically constructs the correct URLs based on `networkPreset` and `specBranch`:
- Config: `https://raw.githubusercontent.com/ethereum/consensus-specs/{specBranch}/configs/{networkPreset}.yaml`
- Presets: 
  - For `minimal`: `https://raw.githubusercontent.com/ethereum/consensus-specs/{specBranch}/presets/minimal/{presetFile}`
  - For `mainnet`: `https://raw.githubusercontent.com/ethereum/consensus-specs/{specBranch}/presets/mainnet/{presetFile}`

## Outputs

The task sets the following output variables:

- `validationSummary`: Object containing:
  - `totalClients`: Total number of clients checked
  - `validClients`: Number of clients that passed validation
  - `invalidClients`: Number of clients that failed validation
  - `results`: Array of validation results for each client

Each result in the `results` array contains:
- `name`: Client name
- `isValid`: Whether the client passed validation
- `missingFields`: Array of required fields that are missing
- `extraFields`: Array of extra fields (if not allowed)
- `errorMessage`: Error message if the client couldn't be checked
- `receivedSpec`: The actual spec returned by the client
- `comparisonIssues`: Array of value mismatches found

## Example Usage

### Basic usage - check all clients (defaults to mainnet)
```yaml
- name: check_consensus_config_spec
  title: "Validate all clients against mainnet spec"
  config:
    clientPattern: ".*"
```

### Check specific clients only
```yaml
- name: check_consensus_config_spec
  title: "Validate Lighthouse clients only"
  config:
    clientPattern: "lighthouse-.*"
```

### Use minimal preset
```yaml
- name: check_consensus_config_spec
  title: "Validate against minimal preset"
  config:
    networkPreset: "minimal"
```

### Use different network config (holesky example with mainnet presets)
```yaml
- name: check_consensus_config_spec
  title: "Validate against holesky config with mainnet presets"
  config:
    # Note: Only mainnet or minimal are valid for networkPreset
    # For other networks like holesky, they use mainnet presets
    networkPreset: "mainnet"
```

### Use master branch instead of dev
```yaml
- name: check_consensus_config_spec
  title: "Validate against master branch specs"
  config:
    specBranch: "master"
```

### Strict validation (no extra fields allowed)
```yaml
- name: check_consensus_config_spec
  title: "Strict spec validation"
  config:
    allowExtraFields: false
```

## Task Behavior

- The task starts by fetching the expected specification from the configured source
- It then continuously polls all matching clients to check their spec endpoint
- For each client, it validates the response against the expected specification
- Results are logged with clear indication of which clients are missing which fields
- The task succeeds only if all clients pass validation

## Error Handling

The task handles various error scenarios:
- Network errors when fetching the specification
- Client timeout or connection errors
- Invalid response format from clients
- Missing or malformed data in client responses

All errors are logged with appropriate context and client identification.