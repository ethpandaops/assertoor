## `run_spamoor_scenario` Task

### Description
The `run_spamoor_scenario` task runs a [spamoor](https://github.com/ethpandaops/spamoor) scenario with the given configuration. Spamoor is a tool for generating various types of Ethereum transactions for testing purposes. This task integrates spamoor scenarios directly into assertoor test workflows, allowing for complex transaction generation patterns.

The task handles wallet management automatically - it creates a wallet pool from the provided private key, initializes the specified scenario, prepares (creates and funds) child wallets as needed, and then executes the scenario.

For a complete list of available scenarios and their configuration options, see the [spamoor transaction scenarios documentation](https://github.com/ethpandaops/spamoor?tab=readme-ov-file#-transaction-scenarios).

### Configuration Parameters

- **`scenarioName`**:\
  The name of the spamoor scenario to run. This must match one of the available scenarios in spamoor (e.g., `eoa-transactions`, `blob-transactions`, `deploy-contracts`).

- **`privateKey`**:\
  The private key of the root wallet used to fund scenario wallets. This wallet should have sufficient ETH to fund all child wallets required by the scenario.

- **`scenarioYaml`**:\
  YAML configuration for the scenario. This is a nested YAML structure that is passed directly to spamoor. The available options depend on the specific scenario being run. See the [spamoor documentation](https://github.com/ethpandaops/spamoor?tab=readme-ov-file#-transaction-scenarios) for scenario-specific configuration options.

### Defaults

Default settings for the `run_spamoor_scenario` task:

```yaml
- name: run_spamoor_scenario
  config:
    scenarioName: ""
    privateKey: ""
    scenarioYaml: null
```

### Example

```yaml
- name: run_spamoor_scenario
  config:
    scenarioName: "eoa-transactions"
    privateKey: "0x1234567890abcdef..."
    scenarioYaml:
      throughput: 10
      max_pending: 100
      max_wallets: 20
```
