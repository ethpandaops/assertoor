# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
coordinator:
  maxConcurrentTests: 1 # max number of tests to run concurrently
  testRetentionTime: 336h # delete test run (logs + status) after that duration

web:
  server:
    host: "0.0.0.0"
    port: 8080
  api:
    enabled: true # enable rest api
  frontend:
    enabled: true # enable web ui

endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

validatorNames:
  inventoryYaml: "./validator-names.yaml"
  inventoryUrl: "https://config.dencun-devnet-12.ethpandaops.io/api/v1/nodes/validator-ranges"
  inventory:
    "0-199": "lighthouse-geth-1"
    "200-399": "teku-besu-1"

globalVars:
  validatorPairNames:
  - "lighthouse-geth-.*"
  - "teku-besu-.*"
  
tests:
# test test1
- name: "Test with local tasks"
  timeout: 48h
  config: {}
  tasks: []
  cleanupTasks: []

externalTests:
- file: ./check-block-proposals2.yaml
  name: "Test with separate list of tasks"
  timeout: 48h
  config: {}

```

- **`coordinator`**:\
  Manages the execution of tests, specifying the maximum number of tests that can run concurrently (`maxConcurrentTests`) and how long to retain test runs, including logs and status, after completion (`testRetentionTime`).

- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web api & frontend, detailing server host and port settings.

- **`validatorNames`**:\
  Defines a mapping of validator index ranges to their respective names. \
  This mapping can be defined directly in the configuration file, imported from an external YAML file, or fetched from a specified URL. \
  These named validator ranges can be referenced in tests for targeted actions and checks.

- **`globalVars`**:\
  A collection of global variables that are made available to all tests and tasks. \
  These variables can be used to maintain consistency and reusability across different test scenarios. \
  For an in-depth explanation, see the "Variables" section of the documentation.

- **`tests`**:\
  A list of tests, each with a specific set of tasks. \
  Every test is identified by a unique name and may include an optional execution timeout. \
  The `config` section within each test allows for the specification of additional variables, which are passed down to the tasks within that test. \
  Moreover, a suite of cleanup tasks can be defined for each test, ensuring orderly and thorough execution, regardless of the test outcome. \
  Detailed guidelines on task configuration can be found in the "Task Configuration" section of the documentation.

- **`externalTests`**:\
  This feature enables the integration of tests that are defined in separate files, fostering a modular and scalable test configuration approach. \
  It allows for better organization and management of complex testing scenarios.

