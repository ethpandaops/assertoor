# Minccino Testnet Testing tool

This project serves as a controller for Ethereum Tests. Users can define tests in yaml, and have the controller ensure the test is conducted while reporting on the status via logs and Prometheus metrics.


### Configuration

## Example Config
```
test:
  name: "basic"

  tasks:
  - name: run_command
    config:
      command:
      - "echo"
      - "hello!"
  - name: execution_is_healthy
  - name: consensus_is_healthy
  - name: both_are_synced
    config:
      consensus:
        percent: 100
      execution:
        percent: 100
  - name: run_command
    config:
      command:
      - "echo"
      - "done!"

execution:
  url: http://localhost:8545

consensus:
  url: http://localhost:5052
```

## Available tasks

```
both_are_synced:
  description: Waits until both consensus and execution clients are considered synced.
  config:
    consensus:
      percent: 100
      wait_for_chain_progression: true
      min_slot_height: 10
    execution:
      percent: 100
      wait_for_chain_progression: true
      min_block_height: 10
consensus_checkpoint_has_progressed:
  description: Checks if a consensus checkpoint has progressed (i.e. if the `head`
    slot has advanced by 3).
  config:
    distance: 3
    checkpoint_name: head
consensus_is_healthy:
  description: Performs a health check against the consensus client.
  config: {}
consensus_is_synced:
  description: Waits until the consensus client considers itself synced.
  config:
    percent: 100
    wait_for_chain_progression: true
    min_slot_height: 10
consensus_is_syncing:
  description: Waits until the consensus client considers itself syncing.
  config: {}
consensus_is_unhealthy:
  description: Performs a health check against the consensus client, finishes when
    the health checks fail.
  config: {}
execution_has_progressed:
  description: Finishes when the execution client has progressed the chain.
  config:
    distance: 3
execution_is_healthy:
  description: Performs a health check against the execution client.
  config: {}
execution_is_synced:
  description: Waits until the execution client considers itself synced.
  config:
    percent: 100
    wait_for_chain_progression: true
    min_block_height: 10
execution_is_unhealthy:
  description: Performs a health check against the execution client. Finishes when
    the execution client is unhealthy.
  config: {}
run_command:
  description: Runs a shell command.
  config:
    command: []
```

