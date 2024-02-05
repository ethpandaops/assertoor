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

#!! cat pkg/coordinator/tasks/*/README.md