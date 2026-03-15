# Assertoor Playbook Examples

Real-world patterns extracted from production playbooks.

---

## Example 1: Block Proposal Check

Verifies every client pair produces at least one block proposal.

```yaml
id: block-proposal-check
name: "Every client pair proposed a block"
timeout: 20m
config:
  validatorPairNames: []

tasks:
- name: check_clients_are_healthy
  title: "Check if at least one client is ready"
  timeout: 5m
  config:
    minClientCount: 1

- name: run_task_matrix
  title: "Check block proposals from all client pairs"
  configVars:
    matrixValues: "validatorPairNames"
  config:
    runConcurrent: true
    matrixVar: "validatorPairName"
    task:
      name: check_consensus_block_proposals
      title: "Wait for block proposal from ${validatorPairName}"
      timeout: 15m
      configVars:
        validatorNamePattern: "validatorPairName"
      config:
        blockCount: 1
```

**Key patterns:**
- Health check gate at the start
- Matrix loop over `validatorPairNames` array
- Concurrent execution for parallel client checking
- `${validatorPairName}` placeholder in title

---

## Example 2: EOA Transaction Testing

Generates transactions in background, verifies inclusion in foreground.

```yaml
id: eoa-transactions-test
name: "Every client proposes blocks with EOA transactions"
timeout: 30m
config:
  walletPrivkey: ""
  validatorPairNames: []

tasks:
- name: check_clients_are_healthy
  title: "Check client health"
  timeout: 5m
  config:
    minClientCount: 1

- name: run_task_background
  title: "Generate and verify EOA transactions"
  config:
    onBackgroundComplete: fail
    backgroundTask:
      name: generate_eoa_transactions
      title: "Generate EOA transactions"
      config:
        childWallets: 100
        limitPending: 100
        limitPerBlock: 10
        limitTotal: 0
        randomTarget: true
        legacyTxType: false
      configVars:
        privateKey: "walletPrivkey"
    foregroundTask:
      name: run_task_matrix
      title: "Check block proposals with transactions"
      configVars:
        matrixValues: "validatorPairNames"
      config:
        runConcurrent: true
        matrixVar: "validatorPairName"
        task:
          name: check_consensus_block_proposals
          title: "Wait for block with >= 5 txs from ${validatorPairName}"
          timeout: 20m
          configVars:
            validatorNamePattern: "validatorPairName"
          config:
            minTransactionCount: 5
```

**Key patterns:**
- Background task for continuous transaction generation
- `onBackgroundComplete: fail` ensures background keeps running
- `limitTotal: 0` means generate indefinitely
- Foreground verifies inclusion across all client pairs

---

## Example 3: Blob Transaction Testing

Tests EIP-4844 blob transaction inclusion.

```yaml
id: blob-transactions-test
name: "Every client proposes blocks with blob transactions"
timeout: 30m
config:
  walletPrivkey: ""
  validatorPairNames: []

tasks:
- name: check_clients_are_healthy
  title: "Check client health"
  timeout: 5m
  config:
    minClientCount: 1

- name: run_task_background
  title: "Generate and verify blob transactions"
  config:
    onBackgroundComplete: fail
    backgroundTask:
      name: generate_blob_transactions
      title: "Generate blob transactions"
      config:
        childWallets: 50
        limitPending: 10
        limitPerBlock: 3
        limitTotal: 0
        blobSidecars: 3
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
          title: "Block from ${validatorPairName} with blobs"
          timeout: 20m
          configVars:
            validatorNamePattern: "validatorPairName"
          config:
            minBlobCount: 1
```

---

## Example 4: Validator Lifecycle (Deposit -> Activate -> Exit)

Complete validator lifecycle test with deposit, activation wait, and exit.

```yaml
id: validator-lifecycle-test
name: "Validator lifecycle: deposit, activate, exit"
timeout: 2h
config:
  walletPrivkey: ""
  depositContract: ""
  validatorCount: 5

tasks:
# Step 1: Generate random mnemonic for new validators
- name: get_random_mnemonic
  title: "Generate validator mnemonic"
  id: gen_mnemonic

# Step 2: Derive public keys
- name: get_pubkeys_from_mnemonic
  title: "Get validator public keys"
  id: gen_pubkeys
  configVars:
    mnemonic: "tasks.gen_mnemonic.outputs.mnemonic"
  config:
    count: 5

# Step 3: Generate deposits
- name: generate_deposits
  title: "Submit deposits"
  id: deposits
  timeout: 30m
  configVars:
    mnemonic: "tasks.gen_mnemonic.outputs.mnemonic"
    walletPrivkey: "walletPrivkey"
    depositContract: "depositContract"
  config:
    limitTotal: 5
    limitPerSlot: 2
    indexCount: 5
    depositAmount: 32000000000
    awaitReceipt: true
    failOnReject: true

# Step 4: Wait for deposits to appear on beacon chain
- name: run_task_matrix
  title: "Verify deposits included"
  configVars:
    matrixValues: "tasks.gen_pubkeys.outputs.pubkeys"
  config:
    runConcurrent: true
    matrixVar: "validatorPubkey"
    task:
      name: check_consensus_block_proposals
      title: "Check deposit for ${validatorPubkey}"
      timeout: 10m
      configVars:
        expectDeposits: "| [.validatorPubkey]"
      config:
        blockCount: 1

# Step 5: Wait for activation
- name: run_task_matrix
  title: "Wait for validator activation"
  timeout: 60m
  configVars:
    matrixValues: "tasks.gen_pubkeys.outputs.pubkeys"
  config:
    runConcurrent: true
    matrixVar: "validatorPubkey"
    task:
      name: check_consensus_validator_status
      title: "Wait for ${validatorPubkey} to activate"
      timeout: 55m
      configVars:
        validatorPubKey: "validatorPubkey"
      config:
        validatorStatus:
          - "active_ongoing"

# Step 6: Get chain specs for exit eligibility calculation
- name: get_consensus_specs
  title: "Get chain specs"
  id: get_specs

# Step 7: Wait for exit eligibility epoch
- name: check_consensus_slot_range
  title: "Wait for exit eligibility"
  configVars:
    minEpochNumber: "| (.tasks.get_specs.outputs.specs.SHARD_COMMITTEE_PERIOD | tonumber) + 5"

# Step 8: Generate voluntary exits
- name: generate_exits
  title: "Submit voluntary exits"
  timeout: 30m
  configVars:
    mnemonic: "tasks.gen_mnemonic.outputs.mnemonic"
  config:
    limitTotal: 5
    limitPerSlot: 2
    indexCount: 5
    awaitInclusion: true

# Step 9: Verify exit status
- name: run_task_matrix
  title: "Verify validators exiting"
  configVars:
    matrixValues: "tasks.gen_pubkeys.outputs.pubkeys"
  config:
    runConcurrent: true
    matrixVar: "validatorPubkey"
    task:
      name: check_consensus_validator_status
      title: "Verify ${validatorPubkey} is exiting"
      timeout: 10m
      configVars:
        validatorPubKey: "validatorPubkey"
      config:
        validatorStatus:
          - "active_exiting"
          - "exited_unslashed"
```

---

## Example 5: Finality and Health Monitoring

Continuous monitoring test for network health.

```yaml
id: network-health-monitor
name: "Network health monitoring"
timeout: 1h
config: {}

tasks:
- name: run_tasks_concurrent
  title: "Monitor network health"
  config:
    tasks:
      - name: check_clients_are_healthy
        title: "Client health"
        timeout: 55m
        config:
          minClientCount: 1
          continueOnPass: true
          failOnCheckMiss: true

      - name: check_consensus_finality
        title: "Finality check"
        timeout: 55m
        config:
          maxUnfinalizedEpochs: 4
          failOnCheckMiss: true
          continueOnPass: true

      - name: check_consensus_attestation_stats
        title: "Attestation quality"
        timeout: 55m
        config:
          minTargetPercent: 80
          minHeadPercent: 70
          minTotalPercent: 90
          failOnCheckMiss: true
          continueOnPass: true
          minCheckedEpochs: 3

      - name: check_consensus_forks
        title: "Fork monitoring"
        timeout: 55m
        config:
          maxForkDistance: 2
          continueOnPass: true

      - name: check_consensus_reorgs
        title: "Reorg monitoring"
        timeout: 55m
        config:
          maxTotalReorgs: 5
          maxReorgDistance: 3
          continueOnPass: true
```

---

## Example 6: EIP-7702 Set Code Transactions

Testing set code (account abstraction) transactions.

```yaml
id: eip7702-test
name: "EIP-7702 set code transaction test"
timeout: 30m
config:
  walletPrivkey: ""

tasks:
- name: check_clients_are_healthy
  title: "Check health"
  timeout: 5m
  config:
    minClientCount: 1

# Deploy a contract to use as delegation target
- name: generate_transaction
  title: "Deploy delegation contract"
  id: deploy
  configVars:
    privateKey: "walletPrivkey"
  config:
    contractDeployment: true
    callData: "0x6080604052..."   # Contract bytecode
    gasLimit: 500000
    awaitReceipt: true
    failOnReject: true

# Create child wallet for 7702 testing
- name: generate_child_wallet
  title: "Create test wallet"
  id: test_wallet
  configVars:
    privateKey: "walletPrivkey"
  config:
    prefundAmount: "1000000000000000000"  # 1 ETH
    randomSeed: true

# Send set code transaction
- name: generate_transaction
  title: "Send EIP-7702 set code tx"
  id: set_code_tx
  configVars:
    privateKey: "walletPrivkey"
  config:
    setCodeTxType: true
    targetAddress: "0x0000000000000000000000000000000000000000"
    gasLimit: 200000
    awaitReceipt: true
    failOnReject: true
    authorizations:
      - chainId: 0
        nonce: 0
        codeAddress: "tasks.deploy.outputs.contractAddress"
        signerPrivkey: "tasks.test_wallet.outputs.childWallet.privateKey"
```

---

## Example 7: EL-Triggered Withdrawal Requests (EIP-7002)

```yaml
id: eip7002-withdrawal-test
name: "EL-triggered withdrawal requests"
timeout: 1h
config:
  walletPrivkey: ""
  validatorMnemonic: ""

tasks:
- name: check_clients_are_healthy
  title: "Check health"
  timeout: 5m
  config:
    minClientCount: 1

# Get validator info
- name: get_pubkeys_from_mnemonic
  title: "Get validator pubkeys"
  id: pubkeys
  configVars:
    mnemonic: "validatorMnemonic"
  config:
    count: 2

# Submit withdrawal requests via EL
- name: generate_withdrawal_requests
  title: "Submit withdrawal requests"
  id: withdrawals
  timeout: 30m
  configVars:
    walletPrivkey: "walletPrivkey"
    sourceMnemonic: "validatorMnemonic"
  config:
    limitTotal: 2
    limitPerSlot: 1
    sourceStartIndex: 0
    sourceIndexCount: 2
    withdrawAmount: 0    # 0 = full exit
    awaitReceipt: true
    failOnReject: true

# Verify withdrawal requests appear in beacon blocks
- name: run_task_matrix
  title: "Verify withdrawal requests in blocks"
  configVars:
    matrixValues: "tasks.pubkeys.outputs.pubkeys"
  config:
    runConcurrent: true
    matrixVar: "validatorPubkey"
    task:
      name: check_consensus_block_proposals
      title: "Check withdrawal request for ${validatorPubkey}"
      timeout: 10m
      configVars:
        expectWithdrawalRequests: "| [{sourceAddress: .depositorAddress, validatorPubkey: .validatorPubkey, amount: 0}]"
      config:
        blockCount: 1
```

---

## Example 8: Slashing Test

```yaml
id: slashing-test
name: "Generate and verify slashings"
timeout: 30m
config:
  walletPrivkey: ""
  slashingMnemonic: ""

tasks:
- name: check_clients_are_healthy
  title: "Check health"
  timeout: 5m
  config:
    minClientCount: 1

- name: get_pubkeys_from_mnemonic
  title: "Get slashable pubkeys"
  id: pubkeys
  configVars:
    mnemonic: "slashingMnemonic"
  config:
    count: 2

# Run slashing generation + verification in parallel
- name: run_task_background
  title: "Slash and verify"
  config:
    onBackgroundComplete: fail
    backgroundTask:
      name: generate_slashings
      title: "Generate attester slashings"
      configVars:
        mnemonic: "slashingMnemonic"
      config:
        slashingType: "attester"
        limitPerSlot: 1
        limitTotal: 2
        indexCount: 2
        awaitInclusion: true
    foregroundTask:
      name: run_task_matrix
      configVars:
        matrixValues: "tasks.pubkeys.outputs.pubkeys"
      config:
        runConcurrent: true
        matrixVar: "validatorPubkey"
        task:
          name: check_consensus_block_proposals
          title: "Verify slashing of ${validatorPubkey}"
          timeout: 20m
          configVars:
            expectSlashings: "| [{publicKey: .validatorPubkey, slashingType: \"attester\"}]"
          config:
            blockCount: 1
```

---

## Example 9: Shell Script Integration

Using `run_shell` for custom logic and variable passing.

```yaml
id: shell-integration-test
name: "Custom shell logic test"
timeout: 15m
config:
  walletPrivkey: ""

tasks:
# Create wallet and capture details
- name: generate_child_wallet
  title: "Create wallet"
  id: wallet
  configVars:
    privateKey: "walletPrivkey"
  config:
    prefundAmount: "500000000000000000"
    randomSeed: true

# Run shell script with environment variables from task outputs
- name: run_shell
  title: "Process wallet data"
  config:
    envVars:
      WALLET_ADDRESS: "tasks.wallet.outputs.childWallet.address"
      WALLET_PRIVKEY: "tasks.wallet.outputs.childWallet.privateKey"
    command: |
      echo "Wallet: $WALLET_ADDRESS"

      # Compute something
      RESULT=$(echo "$WALLET_ADDRESS" | cut -c1-10)

      # Pass computed value back to assertoor as task output
      echo "::set-output shortAddress $RESULT"

      # Set a JSON variable
      echo "::set-json walletInfo {\"address\": \"$WALLET_ADDRESS\", \"short\": \"$RESULT\"}"

      # Set success
      exit 0
```

---

## Example 10: Conditional Test with Setup Phase

```yaml
id: conditional-setup-test
name: "Test with optional setup phase"
timeout: 1h
config:
  runSetup: true
  walletPrivkey: ""
  validatorMnemonic: ""
  existingPubkeys: []

tasks:
# Conditional setup: only run if runSetup is true
- name: run_tasks
  title: "Setup phase"
  if: "runSetup"
  config:
    tasks:
      - name: get_random_mnemonic
        title: "Generate mnemonic"
        id: new_mnemonic

      - name: get_pubkeys_from_mnemonic
        title: "Derive pubkeys"
        id: new_pubkeys
        configVars:
          mnemonic: "tasks.new_mnemonic.outputs.mnemonic"
        config:
          count: 5

      - name: generate_deposits
        title: "Submit deposits"
        configVars:
          mnemonic: "tasks.new_mnemonic.outputs.mnemonic"
          walletPrivkey: "walletPrivkey"
        config:
          limitTotal: 5
          limitPerSlot: 2
          indexCount: 5
          depositAmount: 32000000000
          awaitInclusion: true

# Use either generated or pre-existing pubkeys
- name: run_shell
  title: "Resolve pubkeys"
  config:
    envVars:
      RUN_SETUP: "runSetup"
      NEW_PUBKEYS: "tasks.new_pubkeys.outputs.pubkeys"
      EXISTING: "existingPubkeys"
    command: |
      if [ "$RUN_SETUP" = "true" ]; then
        echo "::set-json activePubkeys $NEW_PUBKEYS"
      else
        echo "::set-json activePubkeys $EXISTING"
      fi

# Verify validators are active
- name: run_task_matrix
  title: "Check validators active"
  configVars:
    matrixValues: "activePubkeys"
  config:
    runConcurrent: true
    matrixVar: "pubkey"
    task:
      name: check_consensus_validator_status
      title: "Check ${pubkey}"
      configVars:
        validatorPubKey: "pubkey"
      config:
        validatorStatus: ["active_ongoing"]
```

---

## Common Configuration Variables

These variables are commonly provided by the assertoor coordinator config and passed to tests:

| Variable | Description | Example |
|----------|-------------|---------|
| `walletPrivkey` | Funded wallet private key for transactions | `"0xdeadbeef..."` |
| `depositContract` | Deposit contract address | `"0x00000000219ab540356cBB839Cbe05303d7705Fa"` |
| `validatorPairNames` | Array of client pair names for matrix tests | `["lighthouse-geth", "prysm-geth"]` |
| `validatorMnemonic` | Mnemonic for validator key derivation | `"abandon abandon..."` |
