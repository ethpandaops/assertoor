## `check_eth_call` Task

### Description
The `check_eth_call` task verifies the response from an `eth_call` transaction on the Ethereum blockchain. This task is essential for validating contract interactions that do not require gas or change the state of the blockchain but need to be tested for expected outcomes.

### Configuration Parameters

- **`ethCallData`**:
  The data to be sent in the eth_call transaction, encoded as a hex string. This typically includes the function signature and arguments for contract interactions.

- **`expectResult`**:
  The expected result of the `eth_call` transaction, expressed as a hex string. This is the value that the call is expected to return under normal circumstances and makes the task succeed.

- **`ignoreResults`**:
  An array of results that, if returned from the `eth_call`, should be ignored. This allows the task to be flexible by acknowledging and skipping known but irrelevant results.

- **`callAddress`**:
  The contract address targeted by the `eth_call`. This should be the address of the contract whose methods are being invoked.

- **`blockNumber`**:
  Specifies the block number at which the state should be queried. A value of `0` typically indicates the latest block.

- **`failOnMismatch`**:
  Determines whether the task should fail if the result of the `eth_call` does not match the `expectResult` and is not in the list of `ignoreResults`. If set to `false`, the task will not fail on a result mismatch, allowing further actions or checks to proceed.

- **`clientPattern`**:
  A regex pattern to select specific client endpoints for sending the `eth_call`. This allows targeting of appropriate nodes within the network.

- **`excludeClientPattern`**:
  A regex pattern to exclude certain clients from being used to make the `eth_call`, optimizing the selection of nodes based on the test scenario.

### Outputs

- **`callResult`**:
  The result of the `eth_call` transaction, returned as a hex string. This output provides direct feedback from the contract method being invoked and is crucial for verifying the result with custom logic.

### Defaults

Default settings for the `check_eth_call` task:

```yaml
- name: check_eth_call
  config:
    ethCallData: "0x"
    expectResult: ""
    ignoreResults: []
    callAddress: "0x0000000000000000000000000000000000000000"
    blockNumber: 0
    failOnMismatch: false
    clientPattern: ""
    excludeClientPattern: ""
```
