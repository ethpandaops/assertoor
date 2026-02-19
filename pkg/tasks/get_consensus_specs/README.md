## `get_consensus_specs` Task

### Description
The `get_consensus_specs` task retrieves the specifications of the consensus chain. This task is crucial for understanding the current parameters and configurations that govern the consensus layer of the Ethereum network.

### Configuration Parameters
This task does not require any specific configuration parameters. It is designed to fetch the consensus chain specifications directly without the need for additional settings.

### Outputs

- **`specs`**:
  This output includes all the specification values of the consensus chain. It provides a comprehensive overview of the network's current operational parameters, such as epoch lengths, reward amounts, slashing penalties, and other critical consensus metrics.

### Defaults

Default settings for the `get_consensus_specs` task:

```yaml
- name: get_consensus_specs
  config: {}
```
