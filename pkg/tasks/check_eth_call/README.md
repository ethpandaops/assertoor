## `check_eth_call` Task

### Description
The `check_eth_call` task verifies the response from an `eth_call` transaction on the Ethereum blockchain. This task is essential for validating contract interactions that do not require gas or change the state of the blockchain but need to be tested for expected outcomes.

#### Task Behavior
- The task makes `eth_call` requests to check contract state.
- By default, the task returns immediately when the expected result is matched.
- Use `continueOnPass: true` to keep monitoring even after success.

### Configuration Parameters

- **`ethCallData`**:\
  The data to be sent in the eth_call transaction, encoded as a hex string. This typically includes the function signature and arguments for contract interactions. Default: `"0x"`.

- **`expectResult`**:\
  The expected result of the `eth_call` transaction, expressed as a hex string. This is the value that the call is expected to return under normal circumstances and makes the task succeed. Default: `""`.

- **`ignoreResults`**:\
  An array of results that, if returned from the `eth_call`, should be ignored. This allows the task to be flexible by acknowledging and skipping known but irrelevant results. Default: `[]`.

- **`callAddress`**:\
  The contract address targeted by the `eth_call`. This should be the address of the contract whose methods are being invoked. Default: `"0x0000000000000000000000000000000000000000"`.

- **`blockNumber`**:\
  Specifies the block number at which the state should be queried. A value of `0` typically indicates the latest block. Default: `0`.

- **`failOnMismatch`**:\
  Determines whether the task should fail if the result of the `eth_call` does not match the `expectResult` and is not in the list of `ignoreResults`. If set to `false`, the task will not fail on a result mismatch. Default: `false`.

- **`clientPattern`**:\
  A regex pattern to select specific client endpoints for sending the `eth_call`. This allows targeting of appropriate nodes within the network. Default: `""`.

- **`excludeClientPattern`**:\
  A regex pattern to exclude certain clients from being used to make the `eth_call`, optimizing the selection of nodes based on the test scenario. Default: `""`.

- **`continueOnPass`**:\
  If set to `true`, the task continues monitoring even after the expected result is matched. This is useful for verifying that the contract state remains consistent over time. If `false` (default), the task exits immediately on success.

### Outputs

- **`callResult`**:\
  The result of the `eth_call` transaction, returned as a hex string. This output provides direct feedback from the contract method being invoked.

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
    continueOnPass: false
```
