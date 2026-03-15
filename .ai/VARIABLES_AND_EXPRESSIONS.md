# Variables and Expression System

## Variable Scoping

### Scope Hierarchy

Variables follow a hierarchical scoping model where child scopes inherit from parent scopes:

```
Global Scope (coordinator globalVars)
  -> Test Scope (created via NewScope from global)
    -> Root Task Scope (inherits test scope)
      -> Child Task Scope (inherits parent task scope)
        -> Grandchild Task Scope (...)
```

### Scope Behavior

- **Reading**: Variables are resolved by checking current scope first, then walking up through parent scopes
- **Writing**: `SetVar()` only modifies the current scope (parent scopes are unaffected)
- **Defaults**: `SetDefaultVar()` sets fallback values with lowest priority
- **Subscopes**: Named namespaces for organizing variables (e.g., `tasks.myTask.outputs`)

### Resolution Order

When resolving a variable:
1. Check for subscope match (e.g., `tasks` prefix routes to tasks subscope)
2. Check current scope's variable map
3. Check parent scope (recursively)
4. Check default variable map
5. Return nil if not found

## Variable Sources

### 1. Global Variables (assertoor config)

```yaml
globalVars:
  walletPrivkey: "0xdeadbeef..."
  depositContract: "0x00000000..."
  validatorPairNames:
    - "lighthouse-geth"
    - "prysm-geth"
```

### 2. Test Config (static defaults)

```yaml
# In test YAML
config:
  walletPrivkey: ""                # Default empty, overridden by global
  depositAmount: 32000000000
  useExistingValidators: false
```

### 3. Test ConfigVars (dynamic from parent)

```yaml
# In test YAML
configVars:
  walletPrivkey: "walletPrivkey"   # Copy from parent scope
```

### 4. Task Outputs

Tasks produce outputs via `ctx.Outputs.SetVar(name, value)`:

```go
// In task implementation
t.ctx.Outputs.SetVar("address", "0x1234...")
t.ctx.Outputs.SetVar("receipt", receiptObject)
```

Outputs are accessible via:
```yaml
configVars:
  myAddr: "tasks.taskId.outputs.address"
```

### 5. Shell Task Variables

The `run_shell` task can set variables via special output patterns:

```bash
# Set a string variable in the task scope
echo "::set-var varName value"

# Set a JSON variable in the task scope
echo '::set-json varName {"key": "value"}'

# Set a task output variable
echo "::set-output outputName value"
```

## Expression System (jq)

### Overview

Assertoor uses the **gojq** library (Go implementation of jq) for expression evaluation. Expressions are used in:
- `configVars` mappings
- `if` conditions on tasks
- Placeholder syntax `${{expression}}`

### Basic Syntax

```yaml
configVars:
  # Simple variable reference (implicitly becomes .walletPrivkey)
  privateKey: "walletPrivkey"

  # Dot-notation navigation
  address: "tasks.wallet.outputs.childWallet.address"

  # Explicit jq with pipe prefix
  calculation: "| .someValue + 1"
```

### Query Normalization

Queries that don't start with `.` or `|` are automatically prefixed with `.`:
- `"walletPrivkey"` becomes `".walletPrivkey"`
- `"tasks.id.outputs.x"` becomes `".tasks.id.outputs.x"`

Queries starting with `|` are prefixed with `. `:
- `"| .x + .y"` becomes `". | .x + .y"`

### Common Expressions

#### Simple References
```yaml
configVars:
  privateKey: "walletPrivkey"                    # -> .walletPrivkey
  mnemonic: "tasks.gen.outputs.mnemonic"         # -> .tasks.gen.outputs.mnemonic
```

#### Array Operations
```yaml
configVars:
  firstFive: "validatorPubkeys[:5]"              # Array slice (first 5)
  afterFive: "validatorPubkeys[5:]"              # Array slice (from index 5)
  firstOne: "validatorPubkeys[0]"                # Single element
  count: "| .validatorPubkeys | length"          # Array length
```

#### Arithmetic
```yaml
configVars:
  nextEpoch: "| .currentEpoch + 1"
  total: "| (.a | tonumber) + (.b | tonumber)"
  # Complex calculation combining multiple task outputs:
  targetEpoch: "| (.tasks.info.outputs.validator.validator.activation_epoch | tonumber) + (.tasks.specs.outputs.specs.SHARD_COMMITTEE_PERIOD | tonumber)"
```

#### Object Construction
```yaml
configVars:
  # Build array of objects
  expectExits: "| [.validatorPubkey]"
  expectBlsChanges: "| [{publicKey: .validatorPubkey, address: .depositorAddress}]"
  expectWithdrawals: "| [{publicKey: .validatorPubkey, address: .address, minAmount: 31000000000}]"
```

#### Filtering
```yaml
configVars:
  activeValidators: "| [.validators[] | select(.status == \"active_ongoing\")]"
  specificEntries: "| [.items[] | select(.key > 5)]"
```

#### String Operations
```yaml
configVars:
  formatted: "| .prefix + \"-\" + .suffix"
```

### If Conditions

The `if` field on tasks evaluates a jq expression. The task is skipped if the result is falsy (false, null, 0, empty string):

```yaml
# Simple boolean check
- name: some_task
  if: "runSetup"
  config: {}

# Comparison
- name: other_task
  if: "| .useExistingValidators == false"
  config: {}

# Complex condition
- name: conditional_task
  if: "| .tasks.check.result == 1"    # Only if check task succeeded
  config: {}
```

### Placeholder Syntax

In string values within YAML, two placeholder syntaxes are supported:

#### Simple: `${varname}`
```yaml
title: "Check block from ${validatorPairName}"
```
Resolves via `LookupVar(varname)` - simple variable lookup.

#### Expression: `${{expression}}`
```yaml
title: "Epoch ${{.currentEpoch}} check"
```
Resolves via `ResolveQuery(expression)` - full jq evaluation.

## ConfigVars Mechanism

The `configVars` field maps configuration struct fields to jq queries:

```yaml
configVars:
  fieldName: "jqQuery"
```

**Processing:**
1. Each query is evaluated against the current variable scope
2. The result is YAML-marshaled then unmarshaled into the target config field
3. Type conversion happens automatically through YAML serialization
4. This runs AFTER static `config` values are applied (overrides them)

**Example:**
```yaml
config:
  limitTotal: 10                    # Static default
  privateKey: ""                    # Placeholder
configVars:
  privateKey: "walletPrivkey"       # Overrides with variable value
  limitTotal: "| .count * 2"       # Overrides with computed value
```

## Type Generalization

Before jq evaluation, Go-typed values are "generalized" through YAML marshal/unmarshal:
- Typed slices (`[]string`, `[]int`) become `[]interface{}`
- Struct fields become `map[string]interface{}`
- This ensures consistent jq behavior regardless of Go source types

## Task Output Namespace

Each task's outputs are stored in an isolated scope accessible via:
```
tasks.<taskId>.outputs.<variableName>
```

Task status is also available:
```
tasks.<taskId>.result          # 0=None, 1=Success, 2=Failure
tasks.<taskId>.running         # bool
tasks.<taskId>.progress        # float64 (0-100)
tasks.<taskId>.progressMessage # string
```

## Scope Control in Flow Tasks

Flow tasks like `run_tasks` and `run_tasks_concurrent` accept a `newVariableScope` option:

- `newVariableScope: false` (default for most) - Child tasks share the parent's scope
- `newVariableScope: true` (default for `run_tasks_concurrent`) - Creates isolated scope

When `newVariableScope: true`, child task variable changes don't leak to siblings or parent.
