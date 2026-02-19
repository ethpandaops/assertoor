## `get_execution_block` Task

## `get_execution_block` Task Documentation

### Description
The `get_execution_block` task is designed to retrieve the most recent block on the chain. This information can be crucial for various purposes, such as tracking chain progress, verifying transactions, or inspecting the block.

### Outputs

- **`header`**:
  The result of the `eth_getBlockByNumber` call, returning the block headers. 

### Defaults
Default settings for the `get_execution_block` task:

```yaml
  - name: get_execution_block
    id: "get_head_block"
    title: "Check head block and block hash from EL RPC"
    timeout: 5m
```