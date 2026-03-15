# Assertoor Task Reference

Complete reference for all assertoor tasks with configuration parameters, output variables, and usage notes.

---

## Flow Control Tasks

### run_tasks

Runs child tasks sequentially. Stops on first failure unless configured otherwise.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `tasks` | array | required | List of task definitions to execute sequentially |
| `newVariableScope` | bool | false | Create isolated variable scope for children |
| `continueOnFailure` | bool | false | Continue executing remaining tasks after a failure |
| `invertResult` | bool | false | Swap success/failure result |
| `ignoreResult` | bool | false | Always report success |

**Outputs:** None

**Example:**
```yaml
- name: run_tasks
  title: "Sequential steps"
  config:
    continueOnFailure: true
    tasks:
      - name: step_one
        title: "First"
        config: {}
      - name: step_two
        title: "Second"
        config: {}
```

---

### run_tasks_concurrent

Runs child tasks in parallel. Configurable success/failure thresholds.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `tasks` | array | required | List of task definitions to execute in parallel |
| `newVariableScope` | bool | true | Create isolated variable scope for children |
| `successThreshold` | uint64 | 0 | Number of tasks that must succeed (0 = all) |
| `failureThreshold` | uint64 | 1 | Number of failures before overall failure |
| `stopOnThreshold` | bool | true | Stop remaining tasks when threshold reached |
| `invertResult` | bool | false | Swap success/failure result |
| `ignoreResult` | bool | false | Always report success |

**Outputs:** None

**Example:**
```yaml
- name: run_tasks_concurrent
  title: "Parallel checks"
  config:
    successThreshold: 0
    tasks:
      - name: check_a
      - name: check_b
```

---

### run_task_matrix

Runs a task template for each value in an array. Supports concurrent or sequential execution.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `task` | object | required | Task template to execute for each matrix value |
| `matrixVar` | string | "" | Variable name to set for each iteration value |
| `matrixValues` | array | [] | Array of values to iterate over |
| `runConcurrent` | bool | false | Run all iterations in parallel |
| `successThreshold` | uint64 | 0 | Required successes (0 = all) |
| `failureThreshold` | uint64 | 1 | Failure limit before stopping |
| `stopOnThreshold` | bool | true | Stop at threshold |
| `invertResult` | bool | false | Swap success/failure |
| `ignoreResult` | bool | false | Always report success |

**Outputs:** None

**Example:**
```yaml
- name: run_task_matrix
  title: "Check all validators"
  configVars:
    matrixValues: "validatorPairNames"
  config:
    runConcurrent: true
    matrixVar: "validatorPairName"
    task:
      name: check_consensus_block_proposals
      title: "Check ${validatorPairName}"
      configVars:
        validatorNamePattern: "validatorPairName"
```

---

### run_task_background

Runs a foreground task and a background task simultaneously. The foreground task determines the overall result.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `foregroundTask` | object | required | Primary task (determines result) |
| `backgroundTask` | object | null | Long-running background task |
| `newVariableScope` | bool | false | Create isolated variable scope |
| `exitOnForegroundSuccess` | bool | false | Stop background when foreground succeeds |
| `exitOnForegroundFailure` | bool | false | Stop background when foreground fails |
| `onBackgroundComplete` | string | "ignore" | Action when background completes: `ignore`, `fail`, `succeed`, `failOrIgnore` |

**Outputs:** None

**onBackgroundComplete options:**
- `ignore` - Background completion has no effect
- `fail` - Fail overall task if background completes (useful: background should run forever)
- `succeed` - Succeed overall task if background completes
- `failOrIgnore` - Fail only if background failed, ignore if it succeeded

**Example:**
```yaml
- name: run_task_background
  config:
    onBackgroundComplete: fail
    backgroundTask:
      name: generate_eoa_transactions
      config:
        limitTotal: 0
    foregroundTask:
      name: check_consensus_block_proposals
      config:
        minTransactionCount: 5
```

---

### run_task_options

Wraps a task with retry, result inversion, and failure handling options.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `task` | object | required | Task to execute |
| `newVariableScope` | bool | false | Create isolated variable scope |
| `retryOnFailure` | bool | false | Retry the task on failure |
| `maxRetryCount` | uint | 0 | Maximum retry attempts (0 = unlimited) |
| `invertResult` | bool | false | Swap success/failure |
| `ignoreResult` | bool | false | Always report success |
| `ignoreFailure` | bool | false | Ignore failure result |
| `expectFailure` | bool | false | Alias for invertResult |

**Outputs:** None

---

### run_external_tasks

Loads and executes a task list from an external YAML file.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `testFile` | string | required | Path or URL to external test YAML file |
| `testConfig` | map | {} | Static configuration values to pass |
| `testConfigVars` | map | {} | Variable mappings to pass |
| `expectFailure` | bool | false | Expect external test to fail |
| `ignoreFailure` | bool | false | Ignore failures from external test |

**Outputs:** None

---

## Check Tasks - Consensus Layer

### check_clients_are_healthy

Monitors health status of consensus and/or execution clients.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientPattern` | string | "" | Regex to select specific clients |
| `pollInterval` | duration | 5s | Interval between health polls |
| `skipConsensusCheck` | bool | false | Skip consensus client checks |
| `skipExecutionCheck` | bool | false | Skip execution client checks |
| `expectUnhealthy` | bool | false | Invert: expect clients to be unhealthy |
| `minClientCount` | int | 0 | Minimum healthy clients required |
| `maxUnhealthyCount` | int | -1 | Max unhealthy clients allowed (-1 = unlimited) |
| `failOnCheckMiss` | bool | false | Fail task when condition not met (vs. keep waiting) |
| `continueOnPass` | bool | false | Keep monitoring after check passes |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `goodClients` | array | Healthy client info objects |
| `failedClients` | array | Unhealthy client info objects |
| `totalCount` | int | Total clients checked |
| `failedCount` | int | Failed client count |
| `goodCount` | int | Healthy client count |

---

### check_consensus_block_proposals

Monitors consensus blocks for ones matching specific criteria. Waits until enough matching blocks are found.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `checkLookback` | int | 1 | Slots to look back for matching blocks |
| `blockCount` | int | 1 | Number of matching blocks required |
| `payloadTimeout` | int | 12 | Seconds to wait for execution payload |
| `graffitiPattern` | string | "" | Regex for block graffiti |
| `validatorNamePattern` | string | "" | Regex for validator name |
| `extraDataPattern` | string | "" | Regex for execution payload extra data |
| `minAttestationCount` | int | 0 | Min attestations in block |
| `minDepositCount` | int | 0 | Min deposits in block |
| `minExitCount` | int | 0 | Min voluntary exits in block |
| `minSlashingCount` | int | 0 | Min total slashings |
| `minAttesterSlashingCount` | int | 0 | Min attester slashings |
| `minProposerSlashingCount` | int | 0 | Min proposer slashings |
| `minBlsChangeCount` | int | 0 | Min BLS to execution changes |
| `minWithdrawalCount` | int | 0 | Min withdrawals |
| `minTransactionCount` | int | 0 | Min transactions |
| `minBlobCount` | int | 0 | Min blob sidecars |
| `minDepositRequestCount` | int | 0 | Min deposit requests (EIP-6110) |
| `minWithdrawalRequestCount` | int | 0 | Min withdrawal requests (EIP-7002) |
| `minConsolidationRequestCount` | int | 0 | Min consolidation requests (EIP-7251) |
| `expectDeposits` | array[string] | [] | Expected validator pubkeys with deposits |
| `expectExits` | array[string] | [] | Expected validator pubkeys with exits |
| `expectSlashings` | array | [] | Expected slashings [{publicKey, slashingType}] |
| `expectBlsChanges` | array | [] | Expected BLS changes [{publicKey, address}] |
| `expectWithdrawals` | array | [] | Expected withdrawals [{publicKey, address, minAmount, maxAmount}] |
| `expectDepositRequests` | array | [] | Expected deposit requests [{publicKey, withdrawalCredentials, amount}] |
| `expectWithdrawalRequests` | array | [] | Expected withdrawal requests [{sourceAddress, validatorPubkey, amount}] |
| `expectConsolidationRequests` | array | [] | Expected consolidations [{sourceAddress, sourcePubkey, targetPubkey}] |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `matchingBlockRoots` | array | Block roots that matched criteria |
| `matchingBlockHeaders` | array | Block headers that matched |
| `matchingBlockBodies` | array | Block bodies that matched |

---

### check_consensus_finality

Monitors chain finality status.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `minUnfinalizedEpochs` | uint64 | 0 | Min unfinalized epochs required |
| `maxUnfinalizedEpochs` | uint64 | 0 | Max unfinalized epochs allowed |
| `minFinalizedEpochs` | uint64 | 0 | Min finalized epochs required |
| `failOnCheckMiss` | bool | false | Fail on condition miss |
| `continueOnPass` | bool | false | Keep monitoring after pass |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `finalizedEpoch` | uint64 | Latest finalized epoch |
| `finalizedRoot` | string | Finalized checkpoint root hash |
| `unfinalizedEpochs` | uint64 | Epochs since last finalized |

---

### check_consensus_forks

Monitors for consensus layer forks.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `minCheckEpochCount` | uint64 | 1 | Min epochs to monitor before evaluating |
| `maxForkDistance` | int64 | 1 | Max allowed fork depth in slots |
| `maxForkCount` | uint64 | 0 | Max forks allowed |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `forks` | array | Fork info objects with head slot, root, clients |

---

### check_consensus_reorgs

Monitors for chain reorganizations.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `minCheckEpochCount` | uint64 | 1 | Min epochs to monitor |
| `maxReorgDistance` | uint64 | 0 | Max reorg depth in slots |
| `maxReorgsPerEpoch` | float64 | 0 | Max average reorgs per epoch |
| `maxTotalReorgs` | uint64 | 0 | Max total reorgs |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:** None

---

### check_consensus_sync_status

Checks consensus clients for sync status.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientPattern` | string | "" | Regex for client selection |
| `pollInterval` | duration | 5s | Poll interval |
| `expectSyncing` | bool | false | Expect clients to be syncing |
| `expectOptimistic` | bool | false | Expect optimistic mode |
| `expectMinPercent` | float64 | 100 | Min % of clients matching condition |
| `expectMaxPercent` | float64 | 100 | Max % of clients matching condition |
| `minSlotHeight` | int | 10 | Min slot height before checking |
| `waitForChainProgression` | bool | false | Wait for chain to progress |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `goodClients` | array | Clients meeting sync criteria |
| `failedClients` | array | Clients not meeting criteria |

---

### check_consensus_validator_status

Checks validator status on the beacon chain.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `validatorPubKey` | string | "" | Validator public key to check |
| `validatorNamePattern` | string | "" | Regex for validator name |
| `validatorIndex` | *uint64 | nil | Validator index to check |
| `validatorStatus` | array[string] | [] | Expected statuses (e.g., `["active_ongoing"]`) |
| `minValidatorBalance` | uint64 | 0 | Min balance in gwei |
| `maxValidatorBalance` | *uint64 | nil | Max balance in gwei |
| `withdrawalCredsPrefix` | string | "" | Expected withdrawal credentials prefix |
| `failOnCheckMiss` | bool | false | Fail on condition miss |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `validator` | object | Full validator information |
| `pubkey` | string | Validator public key |

**Validator status values:** `pending_initialized`, `pending_queued`, `active_ongoing`, `active_exiting`, `active_slashed`, `exited_unslashed`, `exited_slashed`, `withdrawal_possible`, `withdrawal_done`

---

### check_consensus_attestation_stats

Monitors attestation statistics per epoch.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `minTargetPercent` | uint64 | 0 | Min correct target vote % |
| `maxTargetPercent` | uint64 | 100 | Max correct target vote % |
| `minHeadPercent` | uint64 | 0 | Min correct head vote % |
| `maxHeadPercent` | uint64 | 100 | Max correct head vote % |
| `minTotalPercent` | uint64 | 0 | Min total attestation % |
| `maxTotalPercent` | uint64 | 100 | Max total attestation % |
| `failOnCheckMiss` | bool | false | Fail on condition miss |
| `minCheckedEpochs` | uint64 | 1 | Min epochs to check first |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `lastCheckedEpoch` | uint64 | Last epoch checked |
| `validatorCount` | uint64 | Active validator count |
| `validatorBalance` | uint64 | Total effective balance |
| `targetVotes` | uint64 | Correct target votes |
| `targetVotesPercent` | float64 | Target vote percentage |
| `headVotes` | uint64 | Correct head votes |
| `headVotesPercent` | float64 | Head vote percentage |
| `totalVotes` | uint64 | Total attestation votes |
| `totalVotesPercent` | float64 | Total attestation percentage |

---

### check_consensus_proposer_duty

Checks for upcoming proposer duties for specific validators.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `validatorNamePattern` | string | "" | Regex for validator name |
| `validatorIndex` | *uint64 | nil | Specific validator index |
| `minSlotDistance` | uint64 | 0 | Min slots from current for duty |
| `maxSlotDistance` | uint64 | 0 | Max slots from current for duty |
| `failOnCheckMiss` | bool | false | Fail on condition miss |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:** None

---

### check_consensus_slot_range

Waits for consensus wallclock to reach a specific slot/epoch range.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `minSlotNumber` | uint64 | 0 | Min slot number required |
| `maxSlotNumber` | uint64 | max | Max slot number allowed |
| `minEpochNumber` | uint64 | 0 | Min epoch number required |
| `maxEpochNumber` | uint64 | max | Max epoch number allowed |
| `failIfLower` | bool | false | Fail immediately if below min |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `genesisTime` | int64 | Genesis timestamp (Unix seconds) |
| `currentSlot` | uint64 | Current wallclock slot |
| `currentEpoch` | uint64 | Current wallclock epoch |

---

### check_consensus_identity

Checks consensus client node identity, including ENR and CGC extraction.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientPattern` | string | required | Regex for client selection |
| `pollInterval` | duration | 10s | Poll interval |
| `minClientCount` | int | 1 | Min clients required |
| `maxFailCount` | int | -1 | Max failed clients allowed (-1 = unlimited) |
| `failOnCheckMiss` | bool | false | Fail on miss |
| `expectCgc` | *uint64 | nil | Expected custody group count |
| `minCgc` | *uint64 | nil | Min CGC |
| `maxCgc` | *uint64 | nil | Max CGC |
| `expectEnrField` | map | nil | Expected ENR field values |
| `expectPeerIdPattern` | string | "" | Regex for peer ID |
| `expectP2pAddressCount` | *int | nil | Expected P2P address count |
| `expectP2pAddressMatch` | string | "" | Regex for P2P addresses |
| `expectSeqNumber` | *uint64 | nil | Expected metadata sequence number |
| `minSeqNumber` | *uint64 | nil | Min sequence number |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `matchingClients` | array | Clients that passed checks |
| `failedClients` | array | Clients that failed checks |
| `totalCount` | int | Total clients checked |
| `matchingCount` | int | Clients that passed |
| `failedCount` | int | Clients that failed |

---

## Check Tasks - Execution Layer

### check_execution_sync_status

Checks execution clients for sync status.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientPattern` | string | "" | Regex for client selection |
| `pollInterval` | duration | 5s | Poll interval |
| `expectSyncing` | bool | false | Expect syncing |
| `expectMinPercent` | float64 | 100 | Min % matching condition |
| `expectMaxPercent` | float64 | 100 | Max % matching condition |
| `minBlockHeight` | int | 10 | Min block height before checking |
| `waitForChainProgression` | bool | false | Wait for chain progress |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `goodClients` | array | Clients meeting criteria |
| `failedClients` | array | Clients not meeting criteria |

---

### check_eth_call

Executes an eth_call and verifies the response.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ethCallData` | string | "" | Hex-encoded call data |
| `expectResult` | string | "" | Expected hex result |
| `ignoreResults` | array[string] | [] | Hex results to ignore |
| `callAddress` | string | "" | Target contract address |
| `blockNumber` | uint64 | 0 | Block number (0 = latest) |
| `failOnMismatch` | bool | false | Fail on result mismatch |
| `clientPattern` | string | "" | Regex for client selection |
| `excludeClientPattern` | string | "" | Regex to exclude clients |
| `continueOnPass` | bool | false | Keep monitoring |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `callResult` | string | eth_call result as hex |

---

### check_eth_config

Verifies all execution clients return matching eth_config (EIP-7910).

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientPattern` | string | "" | Regex for client selection |
| `excludeClientPattern` | string | "" | Regex to exclude clients |
| `failOnMismatch` | bool | true | Fail when configs don't match |
| `excludeSyncingClients` | bool | false | Exclude syncing clients |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `ethConfig` | string | eth_config JSON from clients |

---

## Generate Tasks - Transactions

### generate_transaction

Sends a single transaction with full control over type and parameters.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `privateKey` | string | required | Wallet private key |
| `legacyTxType` | bool | false | Use legacy transaction type |
| `blobTxType` | bool | false | Use blob transaction (EIP-4844) |
| `setCodeTxType` | bool | false | Use set code transaction (EIP-7702) |
| `blobFeeCap` | *big.Int | nil | Max blob fee cap (wei) |
| `feeCap` | *big.Int | 100 Gwei | Max fee cap (wei) |
| `tipCap` | *big.Int | 1 Gwei | Max priority tip (wei) |
| `gasLimit` | uint64 | 50000 | Gas limit |
| `targetAddress` | string | "" | Target address |
| `randomTarget` | bool | false | Random target address |
| `contractDeployment` | bool | false | Deploy contract |
| `callData` | string | "" | Hex call data |
| `blobData` | string | "" | Hex blob data |
| `blobSidecars` | uint64 | 1 | Number of blob sidecars |
| `randomAmount` | bool | false | Random amount |
| `amount` | *big.Int | 0 | Amount in wei |
| `nonce` | *uint64 | nil | Custom nonce |
| `authorizations` | array | [] | EIP-7702 authorizations [{chainId, nonce, codeAddress, signerPrivkey}] |
| `clientPattern` | string | "" | Regex for client selection |
| `excludeClientPattern` | string | "" | Regex to exclude clients |
| `awaitReceipt` | bool | true | Wait for receipt |
| `failOnReject` | bool | false | Fail on rejection |
| `failOnSuccess` | bool | false | Fail on success (negative testing) |
| `expectEvents` | array | [] | Expected events [{topic0, topic1, topic2, data}] |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `transaction` | object | Transaction object |
| `transactionHex` | string | Transaction hex encoding |
| `transactionHash` | string | Transaction hash |
| `contractAddress` | string | Deployed contract address |
| `receipt` | object | Transaction receipt |

---

### generate_eoa_transactions

Generates multiple EOA transactions continuously.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limitPerBlock` | int | required | Max transactions per block |
| `limitTotal` | int | required | Total limit (0 = unlimited) |
| `limitPending` | int | required | Max pending before waiting |
| `privateKey` | string | required | Wallet private key |
| `childWallets` | uint64 | 0 | Child wallets for parallel sending |
| `walletSeed` | string | "" | Deterministic child wallet seed |
| `refillPendingLimit` | uint64 | 200 | Max pending refill transactions |
| `refillFeeCap` | *big.Int | 500 Gwei | Refill fee cap |
| `refillTipCap` | *big.Int | 1 Gwei | Refill tip cap |
| `refillAmount` | *big.Int | 1 ETH | Refill amount |
| `refillMinBalance` | *big.Int | 0.5 ETH | Min balance before refill |
| `legacyTxType` | bool | false | Legacy transaction type |
| `feeCap` | *big.Int | 100 Gwei | Fee cap |
| `tipCap` | *big.Int | 1 Gwei | Tip cap |
| `gasLimit` | uint64 | 50000 | Gas limit |
| `targetAddress` | string | "" | Target address |
| `randomTarget` | bool | false | Random targets |
| `contractDeployment` | bool | false | Deploy contracts |
| `callData` | string | "" | Call data |
| `randomAmount` | bool | false | Random amounts |
| `amount` | *big.Int | 0 | Amount per transaction |
| `awaitReceipt` | bool | false | Wait for receipts |
| `failOnReject` | bool | false | Fail on rejection |
| `failOnSuccess` | bool | false | Fail on success |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |

**Outputs:** None

---

### generate_blob_transactions

Generates blob transactions (EIP-4844) continuously.

**Config:**
Same as `generate_eoa_transactions` plus:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `blobSidecars` | uint64 | 0 | Blobs per transaction |
| `blobFeeCap` | *big.Int | 10 Gwei | Max blob fee cap |
| `blobData` | string | "" | Hex blob data |
| `legacyBlobTx` | bool | false | Legacy blob format |

**Outputs:** None

---

## Generate Tasks - Validator Operations

### generate_deposits

Generates staking deposits.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limitPerSlot` | int | required | Max deposits per slot |
| `limitTotal` | int | required | Total deposit limit |
| `limitPending` | int | 0 | Max pending deposits |
| `mnemonic` | string | required | Validator key mnemonic |
| `startIndex` | int | 0 | Start index in mnemonic |
| `indexCount` | int | required | Number of validator keys |
| `publicKey` | string | "" | Existing validator pubkey (for top-up) |
| `walletPrivkey` | string | required | Funding wallet private key |
| `depositContract` | string | required | Deposit contract address |
| `depositAmount` | uint64 | 0 | ETH to deposit per validator |
| `depositTxFeeCap` | int64 | 100 Gwei | Deposit tx fee cap |
| `depositTxTipCap` | int64 | 1 Gwei | Deposit tx tip cap |
| `withdrawalCredentials` | string | "" | Custom withdrawal credentials |
| `topUpDeposit` | bool | false | Top up existing validator |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `awaitReceipt` | bool | false | Wait for receipts |
| `failOnReject` | bool | false | Fail on rejection |
| `awaitInclusion` | bool | false | Wait for beacon inclusion |

**Outputs:** None

---

### generate_exits

Generates voluntary validator exits.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limitPerSlot` | int | required | Max exits per slot |
| `limitTotal` | int | required | Total exit limit |
| `mnemonic` | string | required | Validator key mnemonic |
| `startIndex` | int | 0 | Start index in mnemonic |
| `indexCount` | int | required | Number of validator keys |
| `exitEpoch` | int64 | -1 | Exit epoch (-1 = current) |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `awaitInclusion` | bool | false | Wait for beacon inclusion |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `exitedValidators` | array | Validator indices submitted for exit |
| `includedExits` | number | Exits included on-chain |

---

### generate_bls_changes

Generates BLS to execution withdrawal credential changes.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limitPerSlot` | int | required | Max changes per slot |
| `limitTotal` | int | required | Total change limit |
| `mnemonic` | string | required | Validator key mnemonic |
| `startIndex` | int | 0 | Start index |
| `indexCount` | int | required | Number of keys |
| `targetAddress` | string | required | New withdrawal address |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `awaitInclusion` | bool | false | Wait for inclusion |

**Outputs:** None

---

### generate_slashings

Generates slashable attestations or proposals.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `slashingType` | string | "attester" | Type: `attester` or `proposer` |
| `limitPerSlot` | int | required | Max slashings per slot |
| `limitTotal` | int | required | Total slashing limit |
| `mnemonic` | string | required | Validator key mnemonic |
| `startIndex` | int | 0 | Start index |
| `indexCount` | int | required | Number of keys |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `awaitInclusion` | bool | false | Wait for inclusion |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `slashedValidators` | array | Validator indices submitted for slashing |
| `includedSlashings` | number | Slashings included on-chain |

---

### generate_withdrawal_requests

Generates EL-triggered withdrawal requests (EIP-7002).

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limitPerSlot` | int | required | Max requests per slot |
| `limitTotal` | int | required | Total request limit |
| `limitPending` | int | 0 | Max pending requests |
| `sourcePubkey` | string | "" | Single validator pubkey |
| `sourceMnemonic` | string | required | Validator key mnemonic |
| `sourceStartIndex` | int | required | Start index in mnemonic |
| `sourceStartValidatorIndex` | *uint64 | required | Starting validator index |
| `sourceIndexCount` | int | 0 | Number of validators |
| `withdrawAmount` | uint64 | 0 | Gwei to withdraw (0 = full exit) |
| `walletPrivkey` | string | required | Wallet private key |
| `withdrawalContract` | string | 0x00...7002 | Withdrawal contract address |
| `txAmount` | *big.Int | 0.001 ETH | ETH to send with request |
| `txFeeCap` | *big.Int | 100 Gwei | Fee cap |
| `txTipCap` | *big.Int | 1 Gwei | Tip cap |
| `txGasLimit` | uint64 | 200000 | Gas limit |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `awaitReceipt` | bool | false | Wait for receipts |
| `failOnReject` | bool | false | Fail on rejection |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `transactionHashes` | array | Transaction hashes |
| `transactionReceipts` | array | Transaction receipts |

---

### generate_consolidations

Generates validator consolidation requests (EIP-7251).

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limitPerSlot` | int | 0 | Max consolidations per slot |
| `limitTotal` | int | required | Total consolidation limit |
| `limitPending` | int | 0 | Max pending consolidations |
| `sourceMnemonic` | string | required | Source validator mnemonic |
| `sourceStartIndex` | int | required | Source start index |
| `sourceStartValidatorIndex` | *uint64 | required | Source validator index |
| `sourceIndexCount` | int | required | Source validator count |
| `targetPublicKey` | string | required | Target validator pubkey |
| `targetValidatorIndex` | *uint64 | required | Target validator index |
| `consolidationEpoch` | *uint64 | nil | Consolidation epoch |
| `walletPrivkey` | string | "" | Wallet private key |
| `consolidationContract` | string | 0x00...7251 | Contract address |
| `txAmount` | *big.Int | 0.5 ETH | ETH to send |
| `txFeeCap` | *big.Int | 100 Gwei | Fee cap |
| `txTipCap` | *big.Int | 1 Gwei | Tip cap |
| `txGasLimit` | uint64 | 200000 | Gas limit |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `awaitReceipt` | bool | false | Wait for receipts |
| `failOnReject` | bool | false | Fail on rejection |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `transactionHashes` | array | Transaction hashes |
| `transactionReceipts` | array | Transaction receipts |

---

### generate_attestations

Generates custom attestations from derived validator keys.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `mnemonic` | string | required | Validator key mnemonic |
| `startIndex` | int | 0 | Start index |
| `indexCount` | int | required | Number of keys |
| `limitTotal` | int | required | Total attestation limit |
| `limitEpochs` | int | required | Epochs to generate for |
| `clientPattern` | string | "" | Client selection regex |
| `excludeClientPattern` | string | "" | Client exclusion regex |
| `lastEpochAttestations` | bool | false | Reference last epoch |
| `sendAllLastEpoch` | bool | false | All attestations with last epoch data |
| `lateHead` | int | 0 | Slots to delay head vote |
| `randomLateHead` | string | "" | Random delay range (min:max) |
| `lateHeadClusterSize` | int | 0 | Cluster size for shared delay |

**Outputs:** None

---

## Wallet & Key Tasks

### generate_child_wallet

Creates a funded child wallet from a parent wallet.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `privateKey` | string | required | Parent wallet private key |
| `walletSeed` | string | "" | Deterministic seed |
| `randomSeed` | bool | false | Random seed |
| `prefundFeeCap` | *big.Int | nil | Prefund fee cap |
| `prefundTipCap` | *big.Int | nil | Prefund tip cap |
| `prefundAmount` | *big.Int | nil | Amount to transfer |
| `prefundMinBalance` | *big.Int | nil | Min balance trigger |
| `keepFunding` | bool | false | Keep funding loop running |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `childWallet` | object | Wallet info {address, privateKey, balance} |

---

### get_wallet_details

Retrieves wallet balance and nonce.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `privateKey` | string | "" | Wallet private key |
| `address` | string | "" | Wallet address (alternative) |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `address` | string | Wallet address |
| `balance` | string | Balance in wei |
| `nonce` | uint64 | Current nonce |
| `summary` | object | Summary object |

---

### get_pubkeys_from_mnemonic

Derives validator public keys from a BIP39 mnemonic.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `mnemonic` | string | required | BIP39 mnemonic |
| `startIndex` | int | 0 | Start index |
| `count` | int | 1 | Number of keys to derive |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `pubkeys` | array | Derived public keys |

---

### get_random_mnemonic

Generates a random BIP39 mnemonic.

**Config:** None

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `mnemonic` | string | Generated mnemonic |

---

## Data Retrieval Tasks

### get_consensus_specs

Retrieves consensus chain specifications.

**Config:** None

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `specs` | object | Full chain specs (SECONDS_PER_SLOT, SLOTS_PER_EPOCH, etc.) |

---

### get_consensus_validators

Retrieves validators matching specified criteria.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `clientPattern` | string | "" | Client selection regex |
| `validatorNamePattern` | string | "" | Validator name regex |
| `validatorStatus` | array[string] | [] | Status filter |
| `minValidatorBalance` | *uint64 | nil | Min balance filter |
| `maxValidatorBalance` | *uint64 | nil | Max balance filter |
| `withdrawalCredsPrefix` | string | "" | Withdrawal creds prefix |
| `minValidatorIndex` | *uint64 | nil | Min validator index |
| `maxValidatorIndex` | *uint64 | nil | Max validator index |
| `maxResults` | int | 100 | Max results |
| `outputFormat` | string | "full" | Format: `full`, `pubkeys`, or `indices` |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `validators` | array | Full validator info (when format=full) |
| `pubkeys` | array | Public keys (when format=pubkeys) |
| `indices` | array | Validator indices (when format=indices) |
| `count` | int | Number of matching validators |

---

### get_execution_block

Gets the latest execution block header.

**Config:** None

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `header` | object | Execution block header |

---

## Utility Tasks

### sleep

Pauses execution for a specified duration.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `duration` | duration | required | Sleep duration (e.g., "5s", "1m", "1h30m") |

**Outputs:** None

---

### run_shell

Executes a shell script with environment variable support.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `shell` | string | "bash" | Shell interpreter |
| `shellArgs` | array[string] | [] | Shell arguments |
| `envVars` | map[string]string | {} | Environment variables (values are configVar-style queries) |
| `command` | string | required | Shell command to execute |

**Special output patterns in command stdout:**
- `::set-var varName value` - Set variable in task scope
- `::set-json varName {"json": "value"}` - Set JSON variable in task scope
- `::set-output outputName value` - Set task output variable

**Outputs:** Dynamic (based on `::set-output` commands)

---

### run_command

Executes a command with arguments.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `command` | array[string] | required | Command and arguments |
| `allowed_to_fail` | bool | false | Allow command failure |

**Outputs:**
| Variable | Type | Description |
|----------|------|-------------|
| `stdout` | string | Combined stdout/stderr |
| `error` | string | Error message if failed |

---

### run_spamoor_scenario

Runs a spamoor stress testing scenario.

**Config:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `scenarioName` | string | required | Spamoor scenario name |
| `privateKey` | string | required | Root wallet private key |
| `scenarioYaml` | object | nil | Scenario YAML configuration |

**Outputs:** None
