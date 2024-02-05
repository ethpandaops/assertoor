# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
After downloading, run the executable with a test configuration file. The command will be similar to the following:
```
./assertoor --config=./test-config.yaml
```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
Use the following commands to clone the Assertoor repository and navigate to its directory:
```
git clone https://github.com/ethpandaops/assertoor.git
cd assertoor
```
2. **Build the Tool**:\
Compile the source code by running:
```
make build
```
After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
Execute Assertoor with a test configuration file:

```
./bin/assertoor --config=./test-config.yaml
```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

```
docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
```

* **View Logs**:\
To follow the container's logs, use:
```
docker logs assertoor --follow
```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
```
docker rm -f assertoor
```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
- name: "node-1"
executionUrl: "http://127.0.0.1:8545"
consensusUrl: "http://127.0.0.1:5052"

web:
server:
host: "0.0.0.0"
port: 8080
frontend:
enabled: true

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


- **`endpoints`**:\
A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
targetAddress: "..."
# ...
configVars:
privateKey: "walletPrivateKey"
# ...
```

- **`name`**:\
The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
The title helps in identifying and differentiating tasks.

- **`timeout`**:\
An optional parameter specifying the maximum duration for task execution. \
If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
A set of specific settings required for running the task. \
The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
This parameter allows for the use of variables in the task configuration. \
In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**:
These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**:
Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**:
These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
After downloading, run the executable with a test configuration file. The command will be similar to the following:
```
./assertoor --config=./test-config.yaml
```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
Use the following commands to clone the Assertoor repository and navigate to its directory:
```
git clone https://github.com/ethpandaops/assertoor.git
cd assertoor
```
2. **Build the Tool**:\
Compile the source code by running:
```
make build
```
After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
Execute Assertoor with a test configuration file:

```
./bin/assertoor --config=./test-config.yaml
```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

```
docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
```

* **View Logs**:\
To follow the container's logs, use:
```
docker logs assertoor --follow
```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
```
docker rm -f assertoor
```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
- name: "node-1"
executionUrl: "http://127.0.0.1:8545"
consensusUrl: "http://127.0.0.1:5052"

web:
server:
host: "0.0.0.0"
port: 8080
frontend:
enabled: true

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


- **`endpoints`**:\
A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
targetAddress: "..."
# ...
configVars:
privateKey: "walletPrivateKey"
# ...
```

- **`name`**:\
The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
The title helps in identifying and differentiating tasks.

- **`timeout`**:\
An optional parameter specifying the maximum duration for task execution. \
If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
A set of specific settings required for running the task. \
The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
This parameter allows for the use of variables in the task configuration. \
In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**:
These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**:
Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**:
These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
After downloading, run the executable with a test configuration file. The command will be similar to the following:
```
./assertoor --config=./test-config.yaml
```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
Use the following commands to clone the Assertoor repository and navigate to its directory:
```
git clone https://github.com/ethpandaops/assertoor.git
cd assertoor
```
2. **Build the Tool**:\
Compile the source code by running:
```
make build
```
After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
Execute Assertoor with a test configuration file:

```
./bin/assertoor --config=./test-config.yaml
```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

```
docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
```

* **View Logs**:\
To follow the container's logs, use:
```
docker logs assertoor --follow
```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
```
docker rm -f assertoor
```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
- name: "node-1"
executionUrl: "http://127.0.0.1:8545"
consensusUrl: "http://127.0.0.1:5052"

web:
server:
host: "0.0.0.0"
port: 8080
frontend:
enabled: true

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


- **`endpoints`**:\
A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
targetAddress: "..."
# ...
configVars:
privateKey: "walletPrivateKey"
# ...
```

- **`name`**:\
The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
The title helps in identifying and differentiating tasks.

- **`timeout`**:\
An optional parameter specifying the maximum duration for task execution. \
If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
A set of specific settings required for running the task. \
The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
This parameter allows for the use of variables in the task configuration. \
In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**:
These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**:
Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**:
These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

#! cat pkg/coordinator/tasks/*/README.md
# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

#! cat pkg/coordinator/tasks/*/README.md
# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

# Installing Assertoor

## Use Executable from Release

Assertoor provides distribution-specific executables for Windows, Linux, and macOS.

1. **Download the Latest Release**:\
   Navigate to the [Releases](https://github.com/ethpandaops/assertoor/releases) page and download the latest version suitable for your operating system.

2. **Run the Executable**:\
   After downloading, run the executable with a test configuration file. The command will be similar to the following:
    ```
    ./assertoor --config=./test-config.yaml
    ```


## Build from Source

If you prefer to build Assertoor from source, ensure you have [Go](https://go.dev/) `>= 1.21` and Make installed on your machine. Assertoor is tested on Debian, but it should work on other operating systems as well.

1. **Clone the Repository**:\
	Use the following commands to clone the Assertoor repository and navigate to its directory:
    ```
    git clone https://github.com/ethpandaops/assertoor.git
    cd assertoor
    ```
2. **Build the Tool**:\
	Compile the source code by running:
	```
    make build
    ```
	After building, the `assertoor` executable can be found in the `bin` folder.

3. **Run Assertoor**:\
	Execute Assertoor with a test configuration file:

    ```
    ./bin/assertoor --config=./test-config.yaml
    ```

## Use Docker Image

Assertoor also offers a Docker image, which can be found at [ethpandaops/assertoor on Docker Hub](https://hub.docker.com/r/ethpandaops/assertoor).

**Available Tags**:

- `latest`: The latest stable release.
- `v1.0.0`: Version-specific images for all releases.
- `master`: The latest `master` branch version (automatically built).
- `master-xxxxxxx`: Commit-specific builds for each `master` commit (automatically built).

**Running with Docker**:

* **Start the Container**:\
To run Assertoor in a Docker container with your test configuration, use the following command:

  ```
  docker run -d --name=assertoor -v $(pwd):/config -p 8080:8080 -it ethpandaops/assertoor:latest --config=/config/test-config.yaml
  ```

* **View Logs**:\
To follow the container's logs, use:
  ```
  docker logs assertoor --follow
  ```

* **Stop the Container**:\
To stop and remove the Assertoor container, execute:
  ```
  docker rm -f assertoor
  ```
# Configuring Assertoor

The Assertoor test configuration file is a crucial component that outlines the structure and parameters of your tests. \
It contains general settings, endpoints, validator names, global variables, and a list of tests to run.

## Structure of the Assertoor Configuration File

The configuration file is structured as follows:

```yaml
endpoints:
  - name: "node-1"
    executionUrl: "http://127.0.0.1:8545"
    consensusUrl: "http://127.0.0.1:5052"

web:
  server:
    host: "0.0.0.0"
    port: 8080
  frontend:
    enabled: true

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


- **`endpoints`**:\
  A list of Ethereum consensus and execution clients. Each endpoint includes URLs for both RPC endpoints and a name for reference in subsequent tests.

- **`web`**:\
  Configurations for the web frontend, detailing server host and port settings.

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
# Task Configuration

Each task in Assertoor is defined with specific parameters to control its execution. Here's a breakdown of how to configure a task:

## Task Structure

A typical task in Assertoor is structured as follows:

```yaml
name: generate_transaction
title: "Send test transaction"
timeout: 5m
config:
  targetAddress: "..."
  # ...
configVars:
  privateKey: "walletPrivateKey"
  # ...
 ```

- **`name`**:\
  The action that the task should perform. It must correspond to a valid name from the list of supported tasks. \
  This can be a simple action like running a shell script (`run_shell`) or a more complex operation (`generate_transaction`). \
  For a comprehensive list of supported tasks, refer to the "Supported Tasks" section of this documentation.

- **`title`**:\
  A descriptive title for the task. It should be unique and meaningful, as it appears on the web frontend and in the logs.\
  The title helps in identifying and differentiating tasks.

- **`timeout`**:\
  An optional parameter specifying the maximum duration for task execution. \
  If the task exceeds this timeout, it is cancelled and marked as a failure. This parameter helps in managing task execution time and resources.

- **`config`**:\
  A set of specific settings required for running the task. \
  The available settings vary depending on the task name and are detailed in the respective task documentation section.

- **`configVars`**:\
  This parameter allows for the use of variables in the task configuration. \
  In the given example, the value of `walletPrivateKey` is assigned to the `privateKey` config setting of the task. \
  To make this work, the `walletPrivateKey` variable must be defined in a higher scope (globalVars / test config) or set by a previous task (effectively allowing reusing results from these tasks).\
  This feature enables dynamic configuration based on predefined or dynamically set variables.

With this structure, Assertoor tasks can be precisely defined and tailored to fit various testing scenarios.\
Some tasks allow defining subtasks within their configuration, which enables nesting and concurrent execution of tasks.\
The next sections will detail the supported tasks and how to effectively utilize the `config` parameters.

# Supported Tasks in Assertoor

Tasks in Assertoor are fundamental building blocks for constructing comprehensive and dynamic test scenarios. They are designed to be small, logical components that can be combined and configured to meet various testing requirements. Understanding the nature of these tasks, their states, results, and categories is key to effectively using Assertoor.

## Task States and Results

When a task is executed, it goes through different states and yields specific results:

**Task States**

1. **`pending`**: The initial state of a task before execution.
2. **`running`**: The state when a task is actively being executed by the task scheduler.
3. **`completed`**: The final state indicating that a task has finished its execution (regardless of success or failure).

**Task Results**

The result of a task indicates its outcome and can change at any point during its execution:

1. **`none`**: The initial result status of a task, indicating that no definitive outcome has been determined yet.
2. **`success`**: Indicates that the task has achieved its intended objective. \
This result can be set at any point during task execution and does not necessarily imply the task has completed.
3. **`failure`**: Indicates that the task encountered an issue or did not meet the specified conditions. \
Similar to `success`, this result can be updated during the task's execution and does not automatically mean the task has completed.

It's important to note that while the task result can be updated at any time during execution, it becomes final once the task completes. After a task has reached its completion state, the result is conclusive and represents the definitive outcome of that task.

## Task Categories

Tasks in Assertoor are generally categorized based on their primary function and usage in the testing process. It's important to note that this categorization is not strict. While many tasks fit into these categories, some tasks may overlap across multiple categories or not fit into any specific category. However, the majority of currently supported tasks in Assertoor follow this schema:

1. **Flow Tasks (`run_task` prefix)**: 
   These tasks are central to structuring and ordering the execution of other tasks. Flow tasks can define subtasks in their configuration, executing them according to specific logic, such as sequential/concurrent or matrix-based execution. They remain in the `running` state while any of their subtasks are active, and move to `completed` once all subtasks are done. The result of flow tasks is derived from the results of the subtasks, following the flow task's logic. This means the overall outcome of a flow task depends on how its subtasks perform and interact according to the predefined flow.

2. **Check Tasks (`check_` prefix)**: 
   Check tasks are designed to monitor the network and verify certain conditions or states. They typically run continuously, updating their result as they progress. A check task completes by its own only when its result is conclusive and won’t change in the future.

3. **Generate Tasks (`generate_` prefix)**: 
   These tasks perform actions on the network, such as sending transactions or executing deposits/exits. Generate tasks remain in the `running` state while performing their designated actions and move to `completed` upon finishing their tasks.

The categorization serves as a general guide to understanding the nature and purpose of different tasks within Assertoor. As the tool evolves, new tasks may be introduced that further expand or blend these categories, offering enhanced flexibility and functionality in network testing.

The following sections will detail individual tasks within these categories, providing insights into their specific functions, configurations, and use cases.

## `check_clients_are_healthy` Task

### Description
The `check_clients_are_healthy` task is designed to ensure the health of specified clients. It verifies if the clients are reachable and synchronized on the same network.

### Configuration Parameters

- **`clientNamePatterns`**:\
  An array of endpoint names to be checked. If left empty, the task checks all clients. Use this to target specific clients for health checks.

- **`pollInterval`**:\
  The interval at which the health check is performed. Set this to define how frequently the task should check the clients' health.

- **`skipConsensusCheck`**:\
  A boolean value that, when set to `true`, skips the health check for consensus clients. Useful if you only want to focus on execution clients.

- **`skipExecutionCheck`**:\
  A boolean value that, when set to `true`, skips the health check for execution clients. Use this to exclusively check the health of consensus clients.

- **`expectUnhealthy`**:\
  A boolean value that inverts the expected result of the health check. When `true`, the task succeeds if the clients are not ready or unhealthy. This can be useful in test scenarios where client unavailability is expected or being tested.

- **`minClientCount`**:\
  The minimum number of clients that must match the `clientNamePatterns` and pass the health checks for the task to succeed. A value of 0 indicates that all matching clients need to pass the health check. Use this to set a threshold for the number of healthy clients required by your test scenario.

### Defaults

These are the default settings for the `check_clients_are_healthy` task:

```yaml
- name: check_clients_are_healthy
  config:
    clientNamePatterns: []
    pollInterval: 5s
    skipConsensusCheck: false
    skipExecutionCheck: false
    expectUnhealthy: false
    minClientCount: 0
 ```## `check_consensus_attestation_stats` Task

### Description
The `check_consensus_attestation_stats` task is designed to monitor attestation voting statistics on the consensus chain, ensuring that voting patterns align with specified criteria.

### Configuration Parameters

- **`minTargetPercent`**:\
  The minimum percentage of correct target votes per checked epoch required for the task to succeed. The range is 0-100%.

- **`maxTargetPercent`**:\
  The maximum allowable percentage of correct target votes per checked epoch for the task to succeed. The range is 0-100%.

- **`minHeadPercent`**:\
  The minimum percentage of correct head votes per checked epoch needed for the task to succeed. The range is 0-100%.

- **`maxHeadPercent`**:\
  The maximum allowable percentage of correct head votes per checked epoch for the task to succeed. The range is 0-100%.

- **`minTotalPercent`**:\
  The minimum overall voting participation per checked epoch in percent needed for the task to succeed. The range is 0-100%.

- **`maxTotalPercent`**:\
  The maximum allowable overall voting participation per checked epoch for the task to succeed. The range is 0-100%.

- **`failOnCheckMiss`**:\
  Determines whether the task should stop with a failure result if a checked epoch does not meet the specified voting ranges. \
  If `false`, the task continues checking subsequent epochs until it succeeds or times out.

- **`minCheckedEpochs`**:\
  The minimum number of consecutive epochs that must pass the check for the task to succeed.

### Defaults

These are the default settings for the `check_consensus_attestation_stats` task:

```yaml
- name: check_consensus_attestation_stats
  config:
    minTargetPercent: 0
    maxTargetPercent: 100
    minHeadPercent: 0
    maxHeadPercent: 100
    minTotalPercent: 0
    maxTotalPercent: 100
    failOnCheckMiss: false
    minCheckedEpochs: 1
```
## `check_consensus_block_proposals` Task

### Description
The `check_consensus_block_proposals` task checks consensus block proposals to make sure they meet certain requirements. It looks at various details of the blocks to confirm they follow the rules or patterns you set.

### Configuration Parameters

- **`blockCount`**:\
  The number of blocks that need to match your criteria for the task to be successful.

- **`graffitiPattern`**:\
  A pattern to match the graffiti on the blocks.

- **`validatorNamePattern`**:\
  A pattern to identify blocks by the names of their validators.

- **`minAttestationCount`**:\
  The minimum number of attestations (votes or approvals) in a block.

- **`minDepositCount`**:\
  The minimum number of deposit actions required in a block.

- **`minExitCount`**:\
  The minimum number of exit operations in a block.

- **`minSlashingCount`**:\
  The minimum total number of slashing events (penalties for bad actions) in a block.

- **`minAttesterSlashingCount`**:\
  The minimum number of attester slashings in a block.

- **`minProposerSlashingCount`**:\
  The minimum number of proposer slashings in a block.

- **`minBlsChangeCount`**:\
  The minimum number of BLS changes in a block.

- **`minWithdrawalCount`**:\
  The minimum number of withdrawal actions in a block.

- **`minTransactionCount`**:\
  The minimum total number of transactions (any type) needed in a block.

- **`minBlobCount`**:\
  The minimum number of blob sidecars (extra data packets) in a block.

### Defaults

These are the default settings for the `check_consensus_block_proposals` task:

```yaml
- name: check_consensus_block_proposals
  config:
    blockCount: 1
    graffitiPattern: ""
    validatorNamePattern: ""
    minAttestationCount: 0
    minDepositCount: 0
    minExitCount: 0
    minSlashingCount: 0
    minAttesterSlashingCount: 0
    minProposerSlashingCount: 0
    minBlsChangeCount: 0
    minWithdrawalCount: 0
    minTransactionCount: 0
    minBlobCount: 0
```
## `check_consensus_finality` Task

### Description
The `check_consensus_finality` task checks the finality status of the consensus chain. Finality in a blockchain context refers to the point where a block's transactions are considered irreversible.

### Configuration Parameters

- **`minUnfinalizedEpochs`**:\
  The minimum number of epochs that are allowed to be not yet finalized.

- **`maxUnfinalizedEpochs`**:\
  The maximum number of epochs that can remain unfinalized before the task fails.

- **`minFinalizedEpochs`**:\
  The minimum number of epochs that must be finalized for the task to be successful.

- **`failOnCheckMiss`**:\
  If set to `true`, the task will stop with a failure result if the finality status does not meet the criteria specified in the other parameters. \
  If `false`, the task will not fail immediately and will continue checking.

### Defaults

These are the default settings for the `check_consensus_finality` task:

```yaml
- name: check_consensus_finality
  config:
    minUnfinalizedEpochs: 0
    maxUnfinalizedEpochs: 0
    minFinalizedEpochs: 0
    failOnCheckMiss: false
```
## `check_consensus_forks` Task

### Description
The `check_consensus_forks` task is designed to check for forks in the consensus layer of the blockchain. Forks occur when there are divergences in the blockchain, leading to two or more competing chains.

### Configuration Parameters

- **`minCheckEpochCount`**:\
  The minimum number of epochs to check for forks. 

- **`maxForkDistance`**:\
  The maximum distance allowed before a divergence in the chain is counted as a fork. \
  The distance is measured by the number of blocks between the heads of the forked chains.

- **`maxForkCount`**:\
  The maximum number of forks that are acceptable. If the number of forks exceeds this limit, the task will coplete with a failure result.

### Defaults

These are the default settings for the `check_consensus_forks` task:

```yaml
- name: check_consensus_forks
  config:
    minCheckEpochCount: 1
    maxForkDistance: 1
    maxForkCount: 0
```
## `check_consensus_proposer_duty` Task

### Description
The `check_consensus_proposer_duty` task is designed to check for a specific proposer duty on the consensus chain. It verifies if a matching validator is scheduled to propose a block within a specified future time frame (slot distance).

### Configuration Parameters

- **`validatorNamePattern`**:\
  A pattern to identify validators by name. This parameter is used to select validators for the duty check based on their names.

- **`validatorIndex`**:\
  The index of a specific validator to be checked. If this is set, the task focuses on the validator with this index. If it is `null`, the task does not filter by a specific validator index.

- **`maxSlotDistance`**:\
  The maximum number of slots (individual time periods in the blockchain) within which the validator is expected to propose a block. The task succeeds if a matching validator is scheduled for block proposal within this slot distance.

- **`failOnCheckMiss`**:\
  This parameter specifies the task's behavior if a matching proposer duty is not found within the `maxSlotDistance`. If set to `false`, the task continues running until it either finds a matching proposer duty or reaches its timeout. If `true`, the task will fail immediately upon not finding a matching duty.

### Defaults

These are the default settings for the `check_consensus_proposer_duty` task:

```yaml
- name: check_consensus_proposer_duty
  config:
    validatorNamePattern: ""
    validatorIndex: null
    maxSlotDistance: 0
    failOnCheckMiss: false
```## `check_consensus_reorgs` Task

### Description
The `check_consensus_reorgs` task is designed to monitor for reorganizations (reorgs) in the consensus layer of the blockchain. Reorgs occur when the blockchain switches to a different chain due to more blocks being added to it, which can be a normal part of blockchain operation or indicate issues.

### Configuration Parameters

- **`minCheckEpochCount`**:\
  The minimum number of epochs to be checked for reorgs. An epoch is a specific period in blockchain time.

- **`maxReorgDistance`**:\
  The maximum allowable distance for a reorg to occur. This is measured in terms of the number of blocks.

- **`maxReorgsPerEpoch`**:\
  The maximum number of reorgs allowed within a single epoch. If this number is exceeded, it could indicate unusual activity on the blockchain.

- **`maxTotalReorgs`**:\
  The total maximum number of reorgs allowed across all checked epochs. Exceeding this number could be a sign of instability in the blockchain.

### Defaults

These are the default settings for the `check_consensus_reorgs` task:

```yaml
- name: check_consensus_reorgs
  config:
    minCheckEpochCount: 1
    maxReorgDistance: 0
    maxReorgsPerEpoch: 0
    maxTotalReorgs: 0
```
## `check_consensus_slot_range` Task

### Description
The `check_consensus_slot_range` task verifies that the current wall clock time on the consensus chain falls within a specified range of slots and epochs. This is important for ensuring that the chain operates within expected time boundaries.

### Configuration Parameters

- **`minSlotNumber`**:\
  The minimum slot number that the consensus wall clock should be at or above. This sets the lower bound for the check.

- **`maxSlotNumber`**:\
  The maximum slot number that the consensus wall clock should not exceed. This sets the upper bound for the slot range.

- **`minEpochNumber`**:\
  The minimum epoch number that the consensus wall clock should be in or above. Similar to the minSlotNumber, this sets a lower limit, but in terms of epochs.

- **`maxEpochNumber`**:\
  The maximum epoch number that the consensus wall clock should not go beyond. This parameter sets the upper limit for the epoch range.

- **`failIfLower`**:\
  A flag that determines the task's behavior if the current wall clock time is below the specified minimum slot or epoch number. If `true`, the task will fail in such cases; if `false`, it will continue without failing.

### Defaults

These are the default settings for the `check_consensus_slot_range` task:

```yaml
- name: check_consensus_slot_range
  config:
    minSlotNumber: 0
    maxSlotNumber: 18446744073709551615
    minEpochNumber: 0
    maxEpochNumber: 18446744073709551615
    failIfLower: false
```
## `check_consensus_sync_status` Task

### Description
The `check_consensus_sync_status` task checks the synchronization status of consensus clients, ensuring they are aligned with the current state of the blockchain network.

### Configuration Parameters

- **`clientNamePatterns`**:\
  Regex patterns for selecting specific consensus clients by name. The default `".*"` targets all clients.

- **`pollInterval`**:\
  The frequency for checking the clients' sync status.

- **`expectSyncing`**:\
  Set to `true` if the clients are expected to be in a syncing state, or `false` if they should be fully synced.

- **`expectOptimistic`**:\
  When `true`, expects clients to be in an optimistic sync state.

- **`expectMinPercent`**:\
  The minimum sync progress percentage required for the task to succeed.

- **`expectMaxPercent`**:\
  The maximum sync progress percentage allowable for the task to succeed.

- **`minSlotHeight`**:\
  The minimum slot height that clients should be synced to.

- **`waitForChainProgression`**:\
  If set to `true`, the task checks for blockchain progression in addition to synchronization status. If `false`, the task solely checks for synchronization status, without waiting for further chain progression.

### Defaults

Default settings for the `check_consensus_sync_status` task:

```yaml
- name: check_consensus_sync_status
  config:
    clientNamePatterns: [".*"]
    pollInterval: 5s
    expectSyncing: false
    expectOptimistic: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minSlotHeight: 10
    waitForChainProgression: false
```
## `check_consensus_validator_status` Task

### Description
The `check_consensus_validator_status` task is focused on verifying the status of validators on the consensus chain. It checks if the validators are in the expected state, as per the specified criteria.

### Configuration Parameters

- **`validatorPubKey`**:\
  The public key of the validator to be checked. If specified, the task will focus on the validator with this public key.

- **`validatorNamePattern`**:\
  A pattern for identifying validators by name. Useful for filtering validators to be checked based on their names.

- **`validatorIndex`**:\
  The index of a specific validator. If set, the task focuses on the validator with this index. If `null`, no filter on validator index is applied.

- **`validatorStatus`**:\
  A list of allowed validator statuses. The task will check if the validator's status matches any of the statuses in this list.

- **`failOnCheckMiss`**:\
  Determines the task's behavior if the validator's status does not match any of the statuses in `validatorStatus`. If `false`, the task will continue running and wait for the validator to match the expected status. If `true`, the task will fail immediately upon a status mismatch.

### Defaults

These are the default settings for the `check_consensus_validator_status` task:

```yaml
- name: check_consensus_validator_status
  config:
    validatorPubKey: ""
    validatorNamePattern: ""
    validatorIndex: null
    validatorStatus: []
    failOnCheckMiss: false
```
## `check_execution_sync_status` Task

### Description
The `check_execution_sync_status` task checks the synchronization status of execution clients in the blockchain network. It ensures that these clients are syncing correctly with the network's current state.

### Configuration Parameters

- **`clientNamePatterns`**:\
  Regular expression patterns for selecting specific execution clients by name. The default pattern `".*"` targets all clients.

- **`pollInterval`**:\
  The interval at which the task checks the clients' sync status. This defines the frequency of the synchronization checks.

- **`expectSyncing`**:\
  Set this to `true` if the clients are expected to be in a syncing state. If `false`, the task expects the clients to be fully synced.

- **`expectMinPercent`**:\
  The minimum expected percentage of synchronization. Clients should be synced at least to this level for the task to succeed.

- **`expectMaxPercent`**:\
  The maximum allowable percentage of synchronization. Clients should not be synced beyond this level for the task to pass.

- **`minBlockHeight`**:\
  The minimum block height that the clients should be synced to. This sets a specific block height requirement for the task.

- **`waitForChainProgression`**:\
  If `true`, the task checks for blockchain progression in addition to the synchronization status. If `false`, it only checks for synchronization without waiting for further chain progression.

### Defaults

These are the default settings for the `check_execution_sync_status` task:

```yaml
- name: check_execution_sync_status
  config:
    clientNamePatterns: [".*"]
    pollInterval: 5s
    expectSyncing: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minBlockHeight: 10
    waitForChainProgression: false
```
## `generate_blob_transactions` Task

### Description
The `generate_blob_transactions` task creates and sends a large number of blob transactions to the network. It's configured to operate under various limits, and at least one limit parameter is necessary for the task to function.

### Configuration Parameters

- **`limitPerBlock`**:\
  The maximum number of blob transactions to generate per block.

- **`limitTotal`**:\
  The total limit on the number of blob transactions to be generated.

- **`limitPending`**:\
  The limit based on the number of pending blob transactions.

- **`privateKey`**:\
  The private key used for transaction generation.

- **`childWallets`**:\
  The number of child wallets to be created and funded. (If 0, send blob transactions directly from privateKey wallet)

- **`walletSeed`**:\
  The seed phrase used for generating child wallets. (Will be used in combination with privateKey to generate unique child wallets that do not collide with other tasks)

- **`refillPendingLimit`**:\
  The maximum number of pending refill transactions allowed. This limit is used to control the refill process for child wallets, ensuring that the number of refill transactions does not exceed this threshold.

- **`refillFeeCap`**:\
  The maximum fee cap for refilling transactions.

- **`refillTipCap`**:\
  The maximum tip cap for refill transactions.

- **`refillAmount`**:\
  The amount to refill in each child wallet.

- **`refillMinBalance`**:\
  The minimum balance required before triggering a refill.

- **`blobSidecars`**:\
  The number of blob sidecars to include in each transaction.

- **`blobFeeCap`**:\
  The fee cap specifically for blob transactions.

- **`feeCap`**:\
  The maximum fee cap for transactions.

- **`tipCap`**:\
  The tip cap for transactions.

- **`gasLimit`**:\
  The gas limit for each transaction.

- **`targetAddress`**:\
  The target address for transactions.

- **`randomTarget`**:\
  If true, transactions are sent to random addresses.

- **`callData`**:\
  Call data to be included in the transactions.

- **`blobData`**:\
  Data for the blob component of the transactions.

- **`randomAmount`**:\
  If true, the transaction amount is randomized, using `amount` as limit.

- **`amount`**:\
  The amount of ETH (in Wei) to be sent in each blob transaction.

- **`clientPattern`**:\
  A regex pattern to select the specific client endpoint for sending transactions. If empty, any endpoint is used.

### Defaults

Default settings for the `generate_blob_transactions` task:

```yaml
- name: generate_blob_transactions
  config:
    limitPerBlock: 0
    limitTotal: 0
    limitPending: 0
    privateKey: ""
    childWallets: 0
    walletSeed: ""
    refillPendingLimit: 200
    refillFeeCap: "500000000000"
    refillTipCap: "1000000000"
    refillAmount: "1000000000000000000"
    refillMinBalance: "500000000000000000"
    blobSidecars: 1
    blobFeeCap: "10000000000"
    feeCap: "100000000000"
    tipCap: "2000000000"
    gasLimit: 100000
    targetAddress: ""
    randomTarget: false
    callData: ""
    blobData: ""
    randomAmount: false
    amount: "0"
    clientPattern: ""
```
## `generate_bls_changes` Task

### Description
The `generate_bls_changes` task is responsible for generating BLS changes and sending these operations to the network. This task is vital for testing the network's ability to handle changes in withdrawal credentials.

### Configuration Parameters

- **`limitPerSlot`**:\
  The maximum number of BLS change operations to generate for each slot. A slot is a specific time interval in blockchain technology.

- **`limitTotal`**:\
  The total limit on the number of BLS change operations to be generated by this task.

- **`mnemonic`**:\
  A mnemonic phrase used to generate validator keys. This is the starting point for creating BLS key changes.

- **`startIndex`**:\
  The index within the mnemonic from which to start generating validator keys. This determines the starting point for key generation.

- **`indexCount`**:\
  The number of validator keys to generate from the mnemonic. This sets how many different validators will have their keys changed.

- **`targetAddress`**:\
  The address to which the validators' withdrawal credentials will be set. This defines the new target for the validators' funds after the BLS key change.

- **`clientPattern`**:\
  A regex pattern to select the specific client endpoint for sending BLS change operations. If empty, any available endpoint is used.

### Defaults

Default settings for the `generate_bls_changes` task:

```yaml
- name: generate_bls_changes
  config:
    limitPerSlot: 0
    limitTotal: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    targetAddress: ""
    clientPattern: ""
```
## `generate_deposits` Task

### Description
The `generate_deposits` task focuses on creating deposit transactions and sending them to the network. This task is crucial for testing how the network handles new deposits.

### Configuration Parameters

- **`limitPerSlot`**:\
  The maximum number of deposit transactions to be generated for each slot.

- **`limitTotal`**:\
  The total limit on the number of deposit transactions that this task will generate.

- **`mnemonic`**:\
  A mnemonic phrase used to generate validator keys. These keys are essential for creating valid deposit transactions.

- **`startIndex`**:\
  The starting index within the mnemonic for generating validator keys. This defines the beginning point for the key generation process.

- **`indexCount`**:\
  The total number of validator keys to generate from the mnemonic. This number determines how many unique deposit transactions will be created.

- **`walletPrivkey`**:\
  The private key of the wallet from which the deposit will be made. This key is crucial for initiating the deposit transaction.

- **`depositContract`**:\
  The address of the deposit contract on the blockchain. This is the destination where the deposit transactions will be sent.

- **`depositTxFeeCap`**:\
  The maximum fee cap for each deposit transaction. This limits the transaction fees for deposit operations.

- **`depositTxTipCap`**:\
  The maximum tip cap for each deposit transaction. This controls the tip or priority fee for each transaction.

- **`clientPattern`**:\
  A regex pattern for selecting a specific client endpoint to send the deposit transactions. If left empty, any available endpoint will be used.

### Defaults

Default settings for the `generate_deposits` task:

```yaml
- name: generate_deposits
  config:
    limitPerSlot: 0
    limitTotal: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    walletPrivkey: ""
    depositContract: ""
    depositTxFeeCap: 100000000000
    depositTxTipCap: 1000000000
    clientPattern: ""
```
## `generate_eoa_transactions` Task

### Description
The `generate_eoa_transactions` task creates and sends standard transactions from End-User Owned Accounts (EOAs) to the network, essential for testing regular transaction processing.
The task is intended for mass transaction generation.

### Configuration Parameters

- **`limitPerBlock`**:\
  The maximum number of transactions to generate per block.

- **`limitTotal`**:\
  The total limit on the number of transactions to be generated.

- **`limitPending`**:\
  The limit based on the number of pending transactions.

- **`privateKey`**:\
  The private key of the main wallet.

- **`childWallets`**:\
  The number of child wallets to be created and funded. (If 0, send blob transactions directly from privateKey wallet)

- **`walletSeed`**:\
  The seed phrase used for generating child wallets. (Will be used in combination with privateKey to generate unique child wallets that do not collide with other tasks)

- **`refillPendingLimit`**:\
  The maximum number of pending refill transactions allowed. This limit is used to control the refill process for child wallets, ensuring that the number of refill transactions does not exceed this threshold.

- **`refillFeeCap`**:\
  The maximum fee cap for refilling transactions.

- **`refillTipCap`**:\
  The maximum tip cap for refill transactions.

- **`refillAmount`**:\
  The amount to refill in each child wallet.

- **`refillMinBalance`**:\
  The minimum balance required before triggering a refill.

- **`legacyTxType`**:\
  Determines whether to use the legacy type for transactions.

- **`feeCap`**:\
  The maximum fee cap for transactions.

- **`tipCap`**:\
  The tip cap for transactions.

- **`gasLimit`**:\
  The gas limit for each transaction.

- **`targetAddress`**:\
  The target address for transactions.

- **`randomTarget`**:\
  If true, transactions are sent to random addresses.

- **`contractDeployment`**:\
  Determines whether the transactions are for contract deployment.

- **`callData`**:\
  Call data included in the transactions.

- **`randomAmount`**:\
  If true, the transaction amount is randomized.

- **`amount`**:\
  The amount of ETH (in wei) to be sent in each transaction.

- **`clientPattern`**:\
  A regex pattern for selecting a specific client endpoint for transaction sending.

### Defaults

Default settings for the `generate_eoa_transactions` task:

```yaml
- name: generate_eoa_transactions
  config:
    limitPerBlock: 0
    limitTotal: 0
    limitPending: 0
    privateKey: ""
    childWallets: 0
    walletSeed: ""
    refillPendingLimit: 200
    refillFeeCap: "500000000000"
    refillTipCap: "1000000000"
    refillAmount: "1000000000000000000"
    refillMinBalance: "500000000000000000"
    legacyTxType: false
    feeCap: "100000000000"
    tipCap: "1000000000"
    gasLimit: 50000
    targetAddress: ""
    randomTarget: false
    contractDeployment: false
    callData: ""
    randomAmount: false
    amount: "0"
    clientPattern: ""
```
## `generate_exits` Task

### Description
The `generate_exits` task is designed to create and send voluntary exit transactions to the network. This task is essential for testing how the network handles the process of validators voluntarily exiting from their responsibilities.

### Configuration Parameters

- **`limitPerSlot`**:\
  The maximum number of exit transactions to generate per slot.

- **`limitTotal`**:\
  The total limit on the number of exit transactions that the task will generate.

- **`mnemonic`**:\
  A mnemonic phrase used for generating the validators' keys involved in the exit transactions.

- **`startIndex`**:\
  The starting index within the mnemonic from which to begin generating validator keys. This sets the initial point for key generation.

- **`indexCount`**:\
  The number of validator keys to generate from the mnemonic, determining how many unique exit transactions will be created.

- **`clientPattern`**:\
  A regex pattern for selecting a specific client endpoint for sending the exit transactions. If left empty, any available endpoint will be used.

### Defaults

Default settings for the `generate_exits` task:

```yaml
- name: generate_exits
  config:
    limitPerSlot: 0
    limitTotal: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    clientPattern: ""
```
## `generate_slashings` Task

### Description
The `generate_slashings` task is designed to create slashing operations for artificially constructed slashable conditions, and send these operations to the network.\
It's important to note that while the slashing operations are sent to the network, the fake attestations or proposals that justify these slashings are never actually broadcasted. This task is vital for testing the network's response to validator misconduct without affecting the actual network operations.

### Configuration Parameters

- **`slashingType`**:\
  Determines the type of slashing to be simulated. Options are `attester` for attestations-related slashing and `proposer` for proposal-related slashing. \
  This setting decides the kind of validator misbehavior being simulated.

- **`limitPerSlot`**:\
  The maximum number of slashing operations to generate per slot.

- **`limitTotal`**:\
  The total limit on the number of slashing operations to be generated by this task.

- **`mnemonic`**:\
  A mnemonic phrase for generating the keys of validators involved in the simulated slashing.

- **`startIndex`**:\
  The index from which to start generating validator keys within the mnemonic sequence.

- **`indexCount`**:\
  The number of validator keys to generate from the mnemonic, indicating how many distinct slashing operations will be created.

- **`clientPattern`**:\
  A regex pattern to select specific client endpoints for sending the slashing operations. If not specified, the task will use any available endpoint.

### Defaults

Default settings for the `generate_slashings` task:

```yaml
- name: generate_slashings
  config:
    slashingType: attester
    limitPerSlot: 0
    limitTotal: 0
    mnemonic: ""
    startIndex: 0
    indexCount: 0
    clientPattern: ""
```
## `generate_transaction` Task

### Description
The `generate_transaction` task creates and sends a single transaction to the network and optionally checks the transaction receipt. This task is useful for testing specific transaction behaviors, including contract deployments, and verifying receipt properties like triggered events.

### Configuration Parameters

- **`privateKey`**:\
  The private key used for generating the transaction.

- **`legacyTxType`**:\
  If `true`, generates a legacy (type 0) transaction. If `false`, a dynamic fee (type 2) transaction is created.

- **`blobTxType`**:\
  If `true`, generates a blob (type 3) transaction. Otherwise, a dynamic fee (type 2) transaction is used.

- **`blobFeeCap`**:\
  The fee cap for blob transactions. Used only if `blobTxType` is `true`.

- **`feeCap`**:\
  The maximum fee cap for the transaction.

- **`tipCap`**:\
  The tip cap for the transaction.

- **`gasLimit`**:\
  The gas limit for the transaction.

- **`targetAddress`**:\
  The target address for the transaction.

- **`randomTarget`**:\
  If `true`, the transaction is sent to a random address.

- **`contractDeployment`**:\
  If `true`, the transaction is for deploying a contract.

- **`callData`**:\
  Call data included in the transaction.

- **`blobData`**:\
  Data for the blob component of the transaction. Used only if `blobTxType` is `true`.

- **`randomAmount`**:\
  If `true`, the transaction amount is randomized.

- **`amount`**:\
  The amount of cryptocurrency to be sent in the transaction.

- **`clientPattern`**:\
  A regex pattern for selecting a specific client endpoint for sending the transaction.

- **`awaitReceipt`**:\
  If `false`, the task succeeds immediately after sending the transaction without waiting for the receipt. If `true`, it waits for the receipt.

- **`failOnReject`**:\
  If `true`, the task fails if the transaction is rejected.

- **`failOnSuccess`**:\
  If `true`, the task fails if the transaction is successful and not rejected.

- **`expectEvents`**:\
  A list of events that the transaction is expected to trigger, specified in a structured object format. Each event object can have the following properties: `topic0`, `topic1`, `topic2`, `topic3`, and `data`. All these properties are optional and expressed as hexadecimal strings (e.g., "0x000..."). The task checks all triggered events against these objects and looks for a match that satisfies all specified properties in any single event. An example event object might look like this:
  
  ```yaml
  - { "topic0": "0x000...", "topic1": "0x000...", "topic2": "0x000...", "topic3": "0x000...", "data": "0x000..." }
  ```

- **`transactionHashResultVar`**:\
  The variable name to store the transaction hash, available for use by subsequent tasks.

- **`contractAddressResultVar`**:\
  The variable name to store the deployed contract address if the transaction was a contract deployment, available for use by subsequent tasks.

### Defaults

Default settings for the `generate_transaction` task:

```yaml
- name: generate_transaction
  config:
    privateKey: ""
    legacyTxType: false
    blobTxType: false
    blobFeeCap: null
    feeCap: "100000000000"
    tipCap: "1000000000"
    gasLimit: 50000
    targetAddress: ""
    randomTarget: false
    contractDeployment: false
    callData: ""
    blobData: ""
    randomAmount: false
    amount: "0"
    clientPattern: ""
    awaitReceipt: true
    failOnReject: false
    failOnSuccess: false
    expectEvents: []
    transactionHashResultVar: ""
    contractAddressResultVar: ""
```
## `run_command` Task

### Description
The `run_command` task is designed to execute a specified shell command. This task is useful for integrating external scripts or commands into the testing workflow.

### Configuration Parameters

- **`allowed_to_fail`**:
  Determines the task's behavior in response to the command's exit status. If set to `false`, the task will not fail even if the command exits with a failure code. This is useful for commands where a non-zero exit status does not necessarily indicate a critical problem.

- **`command`**:
  The command to be executed, along with its arguments. This should be provided as an array, where the first element is the command and subsequent elements are the arguments. For example, to list files in the current directory with detailed information, you would use `["ls", "-la", "."]`.

### Defaults

Default settings for the `run_command` task:

```yaml
- name: run_command
  config:
    allowed_to_fail: false
    command: []
```
## `run_shell` Task

### Description
The `run_shell` task executes a series of commands within a shell environment. This task is especially useful for complex operations that require executing a shell script or a sequence of commands within a specific shell.


### Configuration Parameters

- **`shell`**:\
  Specifies the type of shell to use for running the commands. Common shells include `sh`, `bash`, `zsh`, etc. The default is `sh`.

- **`envVars`**:\
  A dictionary specifying the environment variables to be used within the shell. Each key in this dictionary represents the name of an environment variable, and its corresponding value indicates the name of a variable from which the actual value should be read. \
  For instance, if `envVars` is set as `{"PATH_VAR": "MY_PATH", "USER_VAR": "MY_USER"}`, the task will use the values of `MY_PATH` and `MY_USER` variables from the current task variable context as the values for `PATH_VAR` and `USER_VAR` within the shell.

- **`command`**:\
  The command or series of commands to be executed in the shell. This can be a single command or a full script. The commands should be provided as a single string, and if multiple commands are needed, they can be separated by the appropriate shell command separator (e.g., newlines or semicolons in `sh` or `bash`).

### Output Handling

The `run_shell` task scans the standard output (stdout) of the shell script for specific triggers that enable actions from within the script. \
For instance, a script can set a variable in the current task context by outputting a formatted string. \
An example of setting a variable `USER_VAR` to "new value" would be:
```bash
echo "::set-var USER_VAR new value"
```
This feature allows the shell script to interact dynamically with the task context, modifying variables based on script execution.

Please note that this feature is still under development and the format of these triggers may change in future versions of the tool. It's important to stay updated with the latest documentation and release notes for any modifications to this functionality.
  

### Defaults

Default settings for the `run_shell` task:

```yaml
- name: run_shell
  config:
    shell: sh
    envVars: {}
    command: ""
```
## `run_task_background` Task

### Description
The `run_task_background` task facilitates the concurrent execution of a foreground task and a background task, with configurable dependencies and outcomes. This task is essential for simulating complex scenarios involving parallel processes.

### Task Behavior
- The task initiates both the foreground and background tasks simultaneously.
- It continuously monitors the status and result of both tasks.
- Upon completion of the foreground task, the `run_task_background` task also completes, and the background task is cancelled if it's still running.
- The result of the `run_task_background` task mirrors the result of the foreground task. For instance, if the foreground task updates its result to "success", the `run_task_background` task will also set its result to "success".
- The behavior following the completion of the background task is configurable based on the `onBackgroundComplete` setting.


### Configuration Parameters

- **`foregroundTask`**:\
  The task that runs in the foreground. This is the primary task and is defined as per the standard task definition format.

- **`backgroundTask`**:\
  The task that runs in the background concurrently with the foreground task. It is also defined following the standard task definition format.

- **`exitOnForegroundSuccess`**:\
  If set to `true`, the `run_task_background` task will exit with a success result when the foreground task's result is set to "success". Note that this does not necessarily mean the foreground task has completed. If still running, both the background and foreground tasks will be cancelled.

- **`exitOnForegroundFailure`**:\
  If `true`, the task exits with a failure result when the foreground task's result is set to "failure". This does not imply the foreground task's completion. Both the background and foreground tasks will be cancelled if they are still running.

- **`onBackgroundComplete`**:\
  Specifies the action to take when the background task completes. Options are:
  - `ignore`: No action is taken.
  - `fail`: Exits the task with a failure result.
  - `succeed`: Exits the task with a success result.
  - `failOrIgnore`: Exits with a failure result if the background task fails, otherwise no action is taken.

- **`newVariableScope`**:\
  Determines if a new variable scope should be created for the foreground task. If `false`, the current scope is passed through. The background task always operates in a new variable scope, which inherits from the parent but does not propagate changes upwards.

### Defaults

Default settings for the `run_task_background` task:

```yaml
- name: run_task_background
  config:
    foregroundTask: {}
    backgroundTask: {}
    exitOnForegroundSuccess: false
    exitOnForegroundFailure: false
    onBackgroundComplete: "ignore"
    newVariableScope: false
```
## `run_task_matrix` Task

### Description
The `run_task_matrix` task is designed to execute a specified task multiple times, each with different input values drawn from an array. This task is ideal for scenarios where you need to test a task under various conditions or with different sets of data.

### Configuration Parameters

- **`runConcurrent`**:\
  Determines whether the child tasks (instances of the task being run for each matrix value) should run concurrently or sequentially. If `true`, all tasks run at the same time; if `false`, they run one after the other.

- **`succeedTaskCount`**:\
  The number of child tasks that need to succeed (result status "success") for the `run_task_matrix` task to stop and return a success result. A value of 0 means all child tasks need to succeed for the overall task to be considered successful.

- **`failTaskCount`**:\
  The number of child tasks that may to fail (result status "failure") before the `run_task_matrix` task to stops and returns a failure result. A value of 0 means that the appearance of any failure in child tasks will cause the overall task to fail.

- **`matrixValues`**:\
  An array of values that form the matrix. Each value in this array is used to run the child task with a different input.

- **`matrixVar`**:\
  The name of the variable to which the current matrix value is assigned for each child task. This allows the child task to access and use the specific value from the matrix.

- **`task`**:\
  The definition of the task to be executed for each matrix value. This task is run repeatedly, once for each value in the `matrixValues` array, with the current value made accessible via the variable named in `matrixVar`.

### Defaults

Default settings for the `run_task_matrix` task:

```yaml
- name: run_task_matrix
  config:
    runConcurrent: false
    succeedTaskCount: 0
    failTaskCount: 0
    matrixValues: []
    matrixVar: ""
    task: {}
```
## `run_task_options` Task

### Description
The `run_task_options` task is designed to execute a single task with configurable behaviors and response actions. This flexibility allows for precise control over how the task's outcome is handled and how it interacts with the overall test environment.

### Configuration Parameters

- **`task`**:\
  The task to be executed. This is defined following the standard task definition format.

- **`exitOnResult`**:\
  If set to `true`, the task will cancel the child task as soon as it sets a result, whether it is "success" or "failure." This option is useful for scenarios where immediate response to the child task's result is necessary.

- **`invertResult`**:\
  When `true`, the result of the child task is inverted. This means the `run_task_options` task will fail if the child task succeeds and succeed if the child task fails. This can be used to validate negative test scenarios.

- **`expectFailure`**:\
  If set to `true`, this option expects the child task to fail. The `run_task_options` task will fail if the child task does not end with a "failure" result, ensuring that failure scenarios are handled as expected.

- **`ignoreFailure`**:\
  When `true`, any failure result from the child task is ignored, and the `run_task_options` task will return a success result instead. This is useful for cases where the child task's failure is an acceptable outcome.

- **`newVariableScope`**:\
  Determines whether to create a new variable scope for the child task. If `false`, the current scope is passed through, allowing the child task to share the same variable context as the `run_task_options` task.

### Defaults

Default settings for the `run_task_options` task:

```yaml
- name: run_task_options
  config:
    task: null
    exitOnResult: false
    invertResult: false
    expectFailure: false
    ignoreFailure: false
    newVariableScope: false
```
## `run_tasks_concurrent` Task

### Description
The `run_tasks_concurrent` task allows for the parallel execution of multiple tasks. This task is crucial in scenarios where tasks need to be run simultaneously, such as in testing environments that require concurrent processes or operations.

### Configuration Parameters

- **`succeedTaskCount`**:\
  The minimum number of child tasks that need to complete with a "success" result for the `run_tasks_concurrent` task to stop and return a success result. A value of 0 indicates that all child tasks need to succeed for the overall task to be considered successful.

- **`failTaskCount`**:\
  The minimum number of child tasks that need to complete with a "failure" result for the `run_tasks_concurrent` task to stop and return a failure result. A value of 1 means the overall task will fail as soon as one child task fails.

- **`tasks`**:\
  An array of child tasks to be executed concurrently. Each task in this array should be defined according to the standard task structure.

### Defaults

Default settings for the `run_tasks_concurrent` task:

```yaml
- name: run_tasks_concurrent
  config:
    succeedTaskCount: 0
    failTaskCount: 1
    tasks: []
```
## `run_tasks` Task

### Description
The `run_tasks` task is designed for executing a series of tasks sequentially, ensuring each task is completed before starting the next. This setup is essential for tests requiring a specific order of task execution.

#### Task Behavior
- The task starts the child tasks one after the other in the order they are listed.
- It continuously monitors the result of the currently running child task. As soon as the child task returns a "success" or "failure" result, the execution of that task is stopped.
- After cancelling the current task, the `run_tasks` task then initiates the next task in the sequence.

An important aspect of this task is that it cancels tasks once they return a result. This is particularly significant for check tasks, which, by their nature, would continue running indefinitely according to their logic. In this sequential setup, however, they are stopped once they achieve a result, allowing the sequence to proceed.

### Configuration Parameters

- **`tasks`**:\
  An array of tasks to be executed one after the other. Each task is defined according to the standard task structure.

- **`expectFailure`**:\
  If set to `true`, this option expects each task in the sequence to fail. The task sequence stops with a "failure" result if any task does not fail as expected.

- **`continueOnFailure`**:\
  When `true`, the sequence of tasks continues even if individual tasks fail, allowing the entire sequence to be executed regardless of individual task outcomes.

### Defaults

Default settings for the `run_tasks` task:

```yaml
- name: run_tasks
  config:
    tasks: []
    expectFailure: false
    continueOnFailure: false
```
## `sleep` Task

### Description
The `sleep` task is designed to introduce a pause or delay in the execution flow for a specified duration. This task is useful in scenarios where a time-based delay is necessary between operations, such as waiting for certain conditions to be met or simulating real-time interactions.

### Configuration Parameters

- **`duration`**:\
  The length of time for which the task should pause execution. The duration is specified in a time format (e.g., '5s' for five seconds, '1m' for one minute). A duration of '0s' means no delay.

### Defaults

Default settings for the `sleep` task:

```yaml
- name: sleep
  config:
    duration: 0s
```
