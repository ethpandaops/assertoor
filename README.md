# Minccino Testnet Testing tool

This project serves as a controller for Ethereum Tests. Users can define tests in yaml, and have the controller ensure the test is conducted while reporting on the status via logs and Prometheus metrics.


### Configuration

## Example Config
```
endpoints:
  - name: "local"
    executionUrl: http://localhost:8545
    consensusUrl: http://localhost:5052

test:
  name: "basic"

  tasks:
  - name: run_command
    config:
      command:
      - "echo"
      - "hello!"
  - name: check_clients_are_healthy
    title: "Check if clients are ready"
    timeout: 30s

  - name: run_tasks_concurrent
    title: "Check if EL & CL clients are synced"
    timeout: 48h
    config:
      tasks:
      - name: check_consensus_sync_status
        title: "Check if CL clients are synced"
      - name: check_execution_sync_status
        title: "Check if EL clients are synced"

  - name: run_command
    config:
      command:
      - "echo"
      - "done!"

```

## Available tasks

```
check_clients_are_healthy:
  description: Checks if clients are healthy.
  config:
    clientNamePatterns:
    - .*
    pollInterval: 5s
    skipConsensusCheck: false
    skipExecutionCheck: false
    expectUnhealthy: false
check_consensus_sync_status:
  description: Checks consensus clients for their sync status.
  config:
    clientNamePatterns:
    - .*
    pollInterval: 5s
    expectSyncing: false
    expectOptimistic: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minSlotHeight: 10
    waitForChainProgression: false
check_execution_sync_status:
  description: Checks execution clients for their sync status.
  config:
    clientNamePatterns:
    - .*
    pollInterval: 5s
    expectSyncing: false
    expectMinPercent: 100
    expectMaxPercent: 100
    minBlockHeight: 10
    waitForChainProgression: false
run_command:
  description: Runs a shell command.
  config:
    allowed_to_fail: false
    command: []
run_tasks:
  description: Run tasks sequentially.
  config:
    tasks: []
run_tasks_concurrent:
  description: Runs multiple tasks in parallel.
  config:
    succeedTaskCount: 0
    failTaskCount: 1
    tasks: []
sleep:
  description: Sleeps for a specified duration.
  config:
    duration: 0s
```

