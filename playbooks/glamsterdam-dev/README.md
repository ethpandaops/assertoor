# glamsterdam-devnet-6 Assertoor Playbooks

These assertoor playbooks cover glamsterdam-devnet-6 (Amsterdam fork) behaviors that
require a live running network and cannot be covered by EELS state/blockchain tests alone:
cross-client consistency, CL/beacon API cross-validation, P2P protocol negotiation,
multi-slot block accounting, and builder lifecycle flows.

EVM semantics EIPs (7708, 8246, 8038, 7954, etc.) are covered by the EELS spec test
suite (`glamsterdam-devnet-6-eels-tests.yaml`) rather than duplicated here.

---

## Playbook Index

| Filename | EIP(s) | What requires a live network |
|----------|--------|------------------------------|
| `glamsterdam-devnet-6-eels-tests.yaml` | 2780, 7708, 7778, 7843, 7928, 7954, 7976, 7981, 7997, 8024, 8037, 8246, 8282 | Runs full EELS spec suite against the live EL client |
| `glamsterdam-devnet-6-eip7778-block-gas.yaml` | EIP-7778 | Verifies `block.gasUsed` diverges from `sum(receipt.gasUsed)` when refund txs are present — requires real mined blocks across multiple slots |
| `glamsterdam-devnet-6-eip7843-slotnum.yaml` | EIP-7843 | Cross-validates SLOTNUM opcode return value against the CL beacon API slot |
| `glamsterdam-devnet-6-eip7928-bal-hash.yaml` | EIP-7928 | Reads `blockAccessListHash` from all EL clients simultaneously to verify cross-client consistency |
| `glamsterdam-devnet-6-eip7975-8159-protocol.yaml` | EIP-7975/8159 | Verifies p2p protocol version negotiation via `admin_peers` — not testable in state tests |
| `glamsterdam-devnet-6-builder-lifecycle.yaml` | EIP-8282 | Builder deposit/exit lifecycle requiring CL slot progression and `execution_requests` in beacon blocks |

---

## Dependencies

### genesis-generator >= 6.1.0

Required for `glamsterdam-devnet-6-eels-tests.yaml` and `glamsterdam-devnet-6-builder-lifecycle.yaml`.
EIP-8282 needs the builder registry predeploy (`0x0000884d2AA32eAa155F59A2f24eFa73D9008282`) in genesis,
which genesis-generator added in 6.1.0. If running on an older generator, skip the builder tests or
run them only after verifying the predeploy exists.

### Foundry (forge + cast)

All playbooks that deploy or interact with contracts install foundry at runtime via `foundryup`.
There is no pre-installed foundry requirement; each test that needs it installs and cleans up its own copy.
An internet connection to `foundry.paradigm.xyz` is required during test execution.

Playbooks that install foundry: `eip7954-initcode`, `eip7778-block-gas`, `eip8037-refund-routing`,
`eip8038-gas-verify`, `eip8246-no-burn`, `builder-lifecycle`, `eip7981-access-list-gas`,
`eip2780-intrinsic-gas`, `eels-tests` (indirectly via foundry steps).

Playbooks that do NOT require foundry: `eip7997-factory`, `eip7843-slotnum`,
`eip7928-bal-hash` (use eth_getBlockByNumber/eth_call via curl only),
`eip8024-opcodes` (uses EELS for the suite; eth_call smoke test via curl).

`eip7708-transfer-logs` installs foundry for Test 3 (deploys a Forwarder contract to
verify internal CALL-with-value also emits Transfer logs). Tests 1 and 2 use curl only.

---

## Running Playbooks: Independent vs. Together

### Run independently (self-contained)

Each glamsterdam-devnet-6 EIP playbook is fully self-contained: it generates its own funded wallets,
installs any tooling it needs, and cleans up after itself. They can be run in any order and in parallel.

| Playbook | Can run in parallel? |
|----------|---------------------|
| `eip7708-transfer-logs` | Yes |
| `eip7843-slotnum` | Yes (depends on EIP-7997 being live in genesis, not on the 7997 playbook) |
| `eip7954-initcode` | Yes |
| `eip7997-factory` | Yes |
| `eip8024-opcodes` | Yes |
| `eip7981-access-list-gas` | Yes |
| `eip2780-intrinsic-gas` | Yes |
| `eip7976-calldata-floor` | Yes |
| `eip7778-block-gas` | Yes |
| `eip7928-bal-hash` | Yes |
| `eip8037-refund-routing` | Yes |
| `eip8038-gas-verify` | Yes |
| `eip8246-no-burn` | Yes |
| `builder-lifecycle` | Yes, but waits for GLOAS fork epoch |

### Run sequentially (ordering recommendation)

If running the full suite manually, a natural order is:

1. `eip7997-factory` — verifies the CREATE2 factory predeploy (other tests may use it)
2. `eip7843-slotnum` — uses the factory internally
3. `eip7708-transfer-logs`, `eip7954-initcode`, `eip8037-refund-routing`, `eip7778-block-gas`, `eip7928-bal-hash`, `eip8038-gas-verify`, `eip8246-no-burn`, `eip8024-opcodes`, `eip7981-access-list-gas`, `eip2780-intrinsic-gas`, `eip7976-calldata-floor` — in any order
4. `builder-lifecycle` — last, since it waits for GLOAS epoch
5. `glamsterdam-devnet-6-eels-tests` — runs the full EELS suite; takes up to 6 hours; run last or standalone

### Legacy devnet playbooks

`bal-devnet-3-eels-tests`, `bal-devnet-4-eels-tests`, `bal-devnet-4-eip8024-stack235-only`,
and `bal-devnet-5-eels-tests` are for previous devnets and should not be run on glamsterdam-devnet-6
(different fork config, different EIP set, pinned to incompatible spec branches).

---

## EIP Reference

| EIP | Title | Amsterdam change |
|-----|-------|-----------------|
| EIP-2780 | Reduce intrinsic transaction gas | TX_BASE=21000 unchanged; calldata token model (zero=4, nonzero=16 gas/byte) |
| EIP-7708 | ETH transfer logs | `Transfer(from, to, value)` emitted by `0xfff...ffe` on every ETH move |
| EIP-7843 | SLOTNUM opcode | New opcode `0x4b` pushes current beacon slot number |
| EIP-7954 | Increase maximum contract sizes | MAX_CODE_SIZE 24576 → 65536; MAX_INIT_CODE_SIZE 49152 → 131072 |
| EIP-7997 | Arachnid CREATE2 factory pre-deploy | Factory at `0x4e59b44847b379578588920ca78fbf26c0b4956c` in genesis |
| EIP-8037 | Source-based gas refund routing | State-clearing refunds credited to tx.origin, not coinbase |
| EIP-8038 | SSTORE gas repricing | COLD_STORAGE_WRITE = 5000 (was 22100); COLD_STORAGE_ACCESS unchanged at 2100 |
| EIP-8246 | SELFDESTRUCT no-burn | ETH sent to address(0) via SELFDESTRUCT is dropped (not credited to address(0)) |
| EIP-7981 | Reduce access list storage key cost | ACCESS_LIST_STORAGE_KEY_COST: 2400 → 1900 (address cost unchanged) |
| EIP-8024 | SWAPN/DUPN/EXCHANGE opcodes | Three new EVM opcodes for stack manipulation; work in legacy bytecode |
| EIP-7778 | Block gas accounting without refunds | block.gasUsed = pre-refund gas; refunds still issued to tx.origin (EIP-8037) but don't reduce block space |
| EIP-7928 | Block-Level Access Lists | Every block header includes `blockAccessListHash`: Keccak256 of RLP-encoded BAL recording all state changes per-tx |
| EIP-7976 | Increase calldata floor cost | floor_data_cost = 21000 + 64 × len(calldata); actual = max(standard, floor) |
| EIP-8282 | Builder execution requests | Builder deposit/exit predeploys; requires genesis-generator >= 6.1.0 |
