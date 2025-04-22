<img align="left" src="./.github/resources/assertoor.png" width="60">
<h1>Assertoor: Ethereum Testnet Testing Tool</h1>

## Overview
Assertoor is a robust and versatile tool designed for comprehensive testing of the Ethereum network. It orchestrates a series of tests from a YAML file, with each test comprising a sequence of tasks executed in a defined order to assess various aspects of the Ethereum network.

## Key Features

- **Connection to Ethereum Clients**:\
  Assertoor connects to multiple Consensus and Execution Clients via their HTTP RPC API, ensuring compatibility with all clients and providing a resilient view of the network status.

- **YAML-Based Test & Task Definition**:\
  Tests, defined and executed through YAML, can include tasks specified in the test configuration or sideloaded from external URLs, offering flexible and organized test management.

- **Task Orchestrator**:\
  Enables execution of tasks in a predefined order, supporting both parallelization and sequential steps with dependencies.

- **Versatile Task Capabilities**:\
  Includes tasks ranging from simple shell scripts to complex built-in logic, such as:
    - **Generating Transactions**: Simulating transaction types to test network response and throughput.
    - **Generating Deposits & Exits**: Evaluating network handling of deposit and exit transactions.
    - **Generating BLS Changes**: Testing network capability to process BLS signature changes.
    - **Checking Network Stability**: Assessing network resilience under various conditions.
    - **Checking Forks & Reorgs**: Analyzing network behavior during forks and reorganizations.
    - **Checking Block Properties**: Testing for specific block properties.
    - ... and more

- **Web Interface for Monitoring**:\
  A user-friendly web interface displays real-time test and task status, logs, and results for easy monitoring and analysis.

- **Web API**:\
  An API interface provides real-time test and task status, logs, and results for easy programmatic access. \
  This feature enables simple integration with other systems and facilitates automated monitoring and analysis workflows.\
  eg. for running [scheduled tests with github workflows](https://github.com/noku-team/assertoor-test)

## Getting Started

1. **Clone the repository & build the tool**:
    ```
    git clone https://github.com/noku-team/assertoor.git
    cd assertoor
    make build
    ```
2. **Configure Your Tests**:\
   Prepare tests in a YAML file. See example configurations [here](https://github.com/noku-team/assertoor/tree/master/example/config). \
  Provide RPC URLs for at least one Client Pair (consensus & execution).
3. **Run Assertoor**:\
   Launch the tool to execute defined tests.
   ```
   ./bin/assertoor --config=./example/config/check_proposals.yaml
   ```
4. **Monitor and Analyze**:\
   Use the web interface to track progress, view logs, and analyze results in real-time.

## Documentation and Examples

Refer to our [documentation](https://github.com/noku-team/assertoor/wiki) for installation, configuration, and usage guidelines. \
Example tests are available [here](https://github.com/noku-team/assertoor/tree/master/playbooks).

## Contributing

Contributions to Assertoor are welcome. Please fork the repository, create a feature branch, and submit a pull request for review.

## License

[![License: GPL-3.0](https://img.shields.io/badge/license-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0) - see the LICENSE file for details.
