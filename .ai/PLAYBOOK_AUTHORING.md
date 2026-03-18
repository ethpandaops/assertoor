# Assertoor Playbook Authoring Guide

## Playbook Structure

Every test playbook is a YAML file with this structure:

```yaml
id: unique-test-identifier
name: "Human-Readable Test Name"
timeout: 30m                    # Max test duration
config:                         # Default variables for this test
  walletPrivkey: ""
  validatorPairNames: []
configVars:                     # Variables copied from parent scope (jq queries)
  someVar: "parentScope.value"
tasks:                          # Main task list (executed sequentially)
  - name: task_type
    title: "Description"
    config: {}
cleanupTasks:                   # Always run after test completes (success or failure)
  - name: cleanup_task
```

## Task Definition

Each task in the `tasks` array has:

```yaml
- name: task_type_name          # Required: registered task name
  title: "Display title"        # Required: shown in logs/UI
  timeout: 5m                   # Optional: task-specific timeout
  id: my_task_id                # Optional: ID for referencing outputs
  if: "condition_expression"    # Optional: jq condition (skip if false)
  config:                       # Task-specific static configuration
    param1: value1
    param2: value2
  configVars:                   # Dynamic variable bindings (key=config field, value=jq query)
    param1: "sourceVariable"
```

## Variable System

### Scoping Hierarchy

```
Global Variables (coordinator config)
  -> Test Scope (test config + configVars)
    -> Task Scope (inherited from test/parent task)
      -> Child Task Scope (inherited from parent task)
```

### Setting Variables

**Static (in config):**
```yaml
config:
  walletPrivkey: "0xabc123..."
  depositAmount: 32000000000
```

**Dynamic (via configVars):**
```yaml
configVars:
  privateKey: "walletPrivkey"                           # Simple variable reference
  address: "tasks.my_wallet.outputs.childWallet.address" # Task output reference
  count: "| .validatorPubkeys | length"                  # jq expression with pipe
```

### Accessing Task Outputs

Tasks with an `id` field expose their outputs to subsequent tasks:

```yaml
- name: generate_child_wallet
  id: my_wallet
  config:
    privateKey: "0x..."

- name: generate_transaction
  configVars:
    # Access output from task "my_wallet"
    privateKey: "tasks.my_wallet.outputs.childWallet.privateKey"
```

### Task Status Variables

Every task exposes status variables accessible via `tasks.<id>`:
- `tasks.<id>.result` - uint8 (0=None, 1=Success, 2=Failure)
- `tasks.<id>.running` - boolean
- `tasks.<id>.progress` - float64 (0-100)
- `tasks.<id>.outputs` - subscope with all output variables

### Placeholder Syntax

In task title values:
- `${varname}` - Simple variable substitution
- `${{.query.path}}` - jq expression evaluation

```yaml
title: "Check block from ${validatorPairName}"
```

### jq Expression Examples

```yaml
configVars:
  # Simple variable reference
  privateKey: "walletPrivkey"

  # Navigate nested objects
  address: "tasks.wallet_task.outputs.childWallet.address"

  # Array slicing
  firstFive: "validatorPubkeys[:5]"
  remaining: "validatorPubkeys[5:]"

  # Pipe expressions (prefix with |)
  epochCalc: "|(.tasks.info.outputs.validator.validator.activation_epoch | tonumber) + 256"

  # Array construction
  expectList: "| [.validatorPubkey]"

  # Complex objects
  expectWithdrawals: "| [{publicKey: .validatorPubkey, address: .depositorAddress, minAmount: 31000000000}]"

  # Conditional/filter
  activeOnes: "| [.validators[] | select(.status == \"active\")]"

  # Length
  count: "| .items | length"
```

## Control Flow Patterns

### Sequential Execution (run_tasks)

```yaml
- name: run_tasks
  title: "Sequential steps"
  config:
    tasks:
      - name: step_one
        title: "First step"
        config: {}
      - name: step_two
        title: "Second step"
        config: {}
```

Options:
- `continueOnFailure: true` - Don't stop on child failure
- `newVariableScope: true` - Isolate variable scope

### Parallel Execution (run_tasks_concurrent)

```yaml
- name: run_tasks_concurrent
  title: "Parallel checks"
  config:
    tasks:
      - name: check_a
        config: {}
      - name: check_b
        config: {}
```

Options:
- `successThreshold: 0` - How many must succeed (0=all)
- `failureThreshold: 1` - How many failures before stopping
- `stopOnThreshold: true` - Stop remaining tasks on threshold

### Matrix/Loop (run_task_matrix)

Execute a task template for each value in an array:

```yaml
- name: run_task_matrix
  title: "Check all validators"
  configVars:
    matrixValues: "validatorPairNames"
  config:
    runConcurrent: true
    matrixVar: "validatorPairName"    # Variable name for current iteration
    task:
      name: check_consensus_block_proposals
      title: "Check ${validatorPairName}"
      configVars:
        validatorNamePattern: "validatorPairName"
      config:
        blockCount: 1
```

Options:
- `runConcurrent: true/false` - Parallel or sequential
- `successThreshold` / `failureThreshold` - Control pass/fail criteria

### Background Tasks (run_task_background)

Run a long-running task in background while foreground task checks results:

```yaml
- name: run_task_background
  title: "Generate and verify transactions"
  config:
    onBackgroundComplete: fail    # fail|ignore|succeed|failOrIgnore
    backgroundTask:
      name: generate_eoa_transactions
      config:
        limitPerBlock: 10
        limitTotal: 1000
        limitPending: 100
        randomTarget: true
      configVars:
        privateKey: "walletPrivkey"
    foregroundTask:
      name: check_consensus_block_proposals
      config:
        minTransactionCount: 5
```

Options:
- `onBackgroundComplete` - What to do when background finishes
- `exitOnForegroundSuccess/Failure` - Control exit conditions

### Conditional Execution

```yaml
- name: some_task
  if: "runSetup == true"
  config: {}

- name: other_task
  if: "| .useExistingValidators == false"
  config: {}
```

### Retry Pattern (run_task_options)

```yaml
- name: run_task_options
  config:
    retryOnFailure: true
    maxRetryCount: 3
    task:
      name: flaky_task
      config: {}
```

Options:
- `retryOnFailure` / `maxRetryCount` - Retry on failure
- `invertResult` / `expectFailure` - Expect task to fail
- `ignoreFailure` / `ignoreResult` - Don't propagate failure

### External Task Files (run_external_tasks)

```yaml
- name: run_external_tasks
  config:
    testFile: "./path/to/other-playbook.yaml"
    testConfig:
      walletPrivkey: ""
    testConfigVars:
      walletPrivkey: "walletPrivkey"
```

## Common Test Patterns

### Pattern 1: Health Check + Block Proposal Verification

```yaml
id: block-proposal-check
name: "Every client pair proposed a block"
timeout: 20m
config:
  validatorPairNames: []

tasks:
- name: check_clients_are_healthy
  title: "Wait for healthy clients"
  timeout: 5m
  config:
    minClientCount: 1

- name: run_task_matrix
  title: "Check proposals from all pairs"
  configVars:
    matrixValues: "validatorPairNames"
  config:
    runConcurrent: true
    matrixVar: "validatorPairName"
    task:
      name: check_consensus_block_proposals
      title: "Wait for block from ${validatorPairName}"
      configVars:
        validatorNamePattern: "validatorPairName"
      config:
        blockCount: 1
```

### Pattern 2: Transaction Generation + Inclusion Verification

```yaml
- name: run_task_background
  title: "Generate and verify tx inclusion"
  config:
    onBackgroundComplete: fail
    backgroundTask:
      name: generate_eoa_transactions
      config:
        childWallets: 100
        limitPending: 100
        limitPerBlock: 10
        limitTotal: 0          # 0 = unlimited
        randomTarget: true
      configVars:
        privateKey: "walletPrivkey"
    foregroundTask:
      name: run_task_matrix
      configVars:
        matrixValues: "validatorPairNames"
      config:
        runConcurrent: true
        matrixVar: "validatorPairName"
        task:
          name: check_consensus_block_proposals
          title: "Block from ${validatorPairName} with >= 5 txs"
          configVars:
            validatorNamePattern: "validatorPairName"
          config:
            minTransactionCount: 5
```

### Pattern 3: Validator Lifecycle (Deposit -> Activate -> Exit)

```yaml
# 1. Generate mnemonic for new validators
- name: get_random_mnemonic
  id: gen_mnemonic

# 2. Get public keys from mnemonic
- name: get_pubkeys_from_mnemonic
  id: gen_pubkeys
  configVars:
    mnemonic: "tasks.gen_mnemonic.outputs.mnemonic"
  config:
    count: 10

# 3. Generate deposits
- name: generate_deposits
  id: deposits
  configVars:
    mnemonic: "tasks.gen_mnemonic.outputs.mnemonic"
  config:
    limitTotal: 10
    limitPerSlot: 2
    indexCount: 10
    depositAmount: 32000000000
    awaitInclusion: true
  configVars:
    walletPrivkey: "walletPrivkey"
    depositContract: "depositContract"

# 4. Wait for activation
- name: run_task_matrix
  configVars:
    matrixValues: "tasks.gen_pubkeys.outputs.pubkeys"
  config:
    runConcurrent: true
    matrixVar: "validatorPubkey"
    task:
      name: check_consensus_validator_status
      title: "Wait for ${validatorPubkey} activation"
      configVars:
        validatorPubKey: "validatorPubkey"
      config:
        validatorStatus: ["active_ongoing"]

# 5. Generate voluntary exits
- name: generate_exits
  configVars:
    mnemonic: "tasks.gen_mnemonic.outputs.mnemonic"
  config:
    limitTotal: 10
    limitPerSlot: 2
    indexCount: 10
    awaitInclusion: true
```

### Pattern 4: Shell Integration

```yaml
- name: run_shell
  title: "Run custom script"
  config:
    envVars:
      WALLET: "tasks.wallet.outputs.address"
    command: |
      echo "Processing wallet: $WALLET"

      # Set task output via magic comments
      echo "::set-output result success"

      # Set task variable (accessible by subsequent tasks)
      echo "::set-var computedValue 42"

      # Set JSON variable
      echo '::set-json myObject {"key": "value", "count": 5}'
```

### Pattern 5: Finality Monitoring

```yaml
- name: check_consensus_finality
  title: "Wait for finality"
  timeout: 30m
  config:
    minFinalizedEpochs: 1
    maxUnfinalizedEpochs: 4
    failOnCheckMiss: true
```

### Pattern 6: Blob Transaction Testing

```yaml
- name: run_task_background
  config:
    onBackgroundComplete: fail
    backgroundTask:
      name: generate_blob_transactions
      config:
        limitPerBlock: 3
        limitTotal: 0
        limitPending: 10
        blobSidecars: 3
      configVars:
        privateKey: "walletPrivkey"
    foregroundTask:
      name: run_task_matrix
      configVars:
        matrixValues: "validatorPairNames"
      config:
        runConcurrent: true
        matrixVar: "validatorPairName"
        task:
          name: check_consensus_block_proposals
          title: "Block from ${validatorPairName} with blobs"
          configVars:
            validatorNamePattern: "validatorPairName"
          config:
            minBlobCount: 1
```

## Best Practices

1. **Always start with a health check** - Use `check_clients_are_healthy` with `minClientCount: 1` before running tests
2. **Use task IDs** for any task whose outputs you need to reference later
3. **Set appropriate timeouts** - Validator operations need longer timeouts (30m+), transaction checks need shorter ones
4. **Use matrix for multi-client testing** - `run_task_matrix` with `runConcurrent: true` over `validatorPairNames`
5. **Background + foreground pattern** for continuous generation with verification
6. **Always include cleanup tasks** for validator lifecycle tests (exit created validators)
7. **Use configVars** for dynamic values, `config` for static values
8. **Descriptive titles** with `${variable}` placeholders for clarity in logs
9. **Chain specs** - Use `get_consensus_specs` to retrieve chain parameters for calculations
10. **Client selection** - Use `clientPattern` regex to target specific clients (e.g., `"lighthouse.*"`)
