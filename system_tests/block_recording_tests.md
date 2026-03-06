# Block Recording Tests for JIT Benchmarking

This document describes each test in `block_recording_test.go`. These tests generate
blocks of varying sizes and complexity for use as inputs to JIT benchmarks. Each test
uses `setupProgramTest` (with JIT enabled) to spin up a local L2 node, executes
transactions, and calls `recordBlock` to persist the block's validation inputs as JSON.

## How it works

Every test follows the same lifecycle:

1. **Setup**: `recordBlockSetup` calls `setupProgramTest(t, true)` which creates a
   full local L2 node with a sequencer, batch poster, and validation node.
   `setupProgramTest` also configures Stylus (ink price, chain owner, etc.).
2. **Execute transactions**: The test sends one or more transactions to the L2
   sequencer. In Arbitrum Nitro the sequencer typically packages each transaction into
   its own L2 block, so the **recorded block contains the single transaction whose
   receipt is passed to `recordBlock`**. Prior transactions in the same test land in
   earlier blocks and are *not* recorded (they exist only to set up state).
3. **Record**: `record(t, blockNum, builder)` writes a JSON file with all data
   required to re-execute that block through the arbitrator prover or JIT binary
   (messages, delayed messages, preimages, WASM modules, etc.).

### Running the tests

```bash
# Record a single test
go test -v -run "TestRecordBlockSingleTransfer$" ./system_tests/... -count 1 -- \
  --recordBlockInputs.enable=true \
  --recordBlockInputs.WithBaseDir=target/ \
  --recordBlockInputs.WithTimestampDirEnabled=false \
  --recordBlockInputs.WithBlockIdInFileNameEnabled=false

# Record all block-recording tests at once
go test -v -run "TestRecordBlock" ./system_tests/... -count 1 -- \
  --recordBlockInputs.enable=true \
  --recordBlockInputs.WithBaseDir=target/ \
  --recordBlockInputs.WithTimestampDirEnabled=false \
  --recordBlockInputs.WithBlockIdInFileNameEnabled=false
```

---

## Test Catalog

### Category 1 — Simple ETH Transfers

These tests produce blocks with only native value-transfer transactions. No contract
code is executed; the EVM touches only balance state. They represent the cheapest
possible blocks.

#### 1. `TestRecordBlockSingleTransfer`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 ETH transfer |
| **EVM opcodes exercised** | None (pure value transfer) |
| **State changes** | 2 balance updates (sender, receiver) |
| **Expected block size** | Tiny |

Generates a single account (`User1`) and transfers 0.01 ETH from `Owner`. The
recorded block is the one containing that single transfer. This is the smallest
possible meaningful block — a baseline for JIT benchmarks.

---

#### 2. `TestRecordBlockMultipleTransfers`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 ETH transfer (the last of 5) |
| **EVM opcodes exercised** | None |
| **State changes** | 2 balance updates |
| **Expected block size** | Tiny |

Sends 5 sequential ETH transfers, each to a different generated account. Transactions
are sent one at a time (send → wait for receipt), so each lands in its own block. The
**last block** is recorded. The recorded block itself is still a single-transfer block,
but the chain state at that point has been modified by 4 prior transfers (warmed-up
state trie).

---

#### 3. `TestRecordBlockManyTransfers`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 ETH transfer (the last of 50) |
| **EVM opcodes exercised** | None |
| **State changes** | 2 balance updates |
| **Expected block size** | Tiny (but with 50 prior state modifications) |

Creates 50 accounts and fires all 50 transfers into the sequencer using
`SendWaitTestTransactions` (sends all txs, then waits for all receipts). The recorded
block is the last receipt's block. Due to the batch-send pattern, multiple transfers
*may* land in the same block if the sequencer batches them, but the sequencer typically
still processes one tx per block. This test is useful for measuring JIT performance
against a chain with many recent state changes in the trie.

---

#### 4. `TestRecordBlockTransfersWithCalldata`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 ETH transfer with 4 KB of random calldata |
| **EVM opcodes exercised** | None (EOA-to-EOA, calldata is ignored) |
| **State changes** | 2 balance updates |
| **Expected block size** | Small–medium (large tx payload) |

Sends 4 transfers to the same EOA with increasing calldata sizes: 32 B, 256 B, 1 KB,
4 KB. Only the last transfer (4 KB calldata) is recorded. Although the EVM does not
execute the calldata (the recipient is an EOA), the calldata is part of the transaction
and therefore part of the block data that the JIT must process. This measures how
serialization / deserialization overhead scales with payload size.

---

### Category 2 — Solidity Contract Deployments

These tests produce blocks where the recorded transaction is a contract-creation
transaction. The EVM executes the constructor (initcode), stores the resulting bytecode,
and creates a new account with code.

#### 5. `TestRecordBlockSolidityDeploy`

| Attribute | Value |
|---|---|
| **Recorded blocks** | 2: deployment of `Simple`, then `Simple.Increment()` call |
| **Contracts deployed** | `Simple` (small Solidity contract) |
| **EVM opcodes exercised** | Constructor execution, SSTORE, SLOAD, event emission |
| **State changes** | Contract creation + code storage; then counter increment |
| **Expected block size** | Small (deploy), tiny (call) |

Deploys the `Simple` contract and records that deployment block. Then calls
`Simple.Increment()` and records that block too. This gives you two recorded blocks:
one with a small contract deployment and one with a minimal Solidity state-mutating
call.

---

#### 6. `TestRecordBlockERC20Deploy`

| Attribute | Value |
|---|---|
| **Recorded block contains** | ERC20 contract deployment |
| **Contracts deployed** | `ERC20` (OpenZeppelin-style, with name/symbol constructor args) |
| **EVM opcodes exercised** | Constructor with string arguments, SSTORE for name/symbol/decimals |
| **State changes** | Contract creation + multiple storage slots |
| **Expected block size** | Medium |

Deploys a full ERC20 token contract ("TestToken" / "TT"). The ERC20 bytecode is
significantly larger than `Simple`, and the constructor writes multiple storage slots
(name, symbol, decimals). This produces a medium-sized deployment block.

---

#### 7. `TestRecordBlockMultipleSolidityDeploys`

| Attribute | Value |
|---|---|
| **Recorded block contains** | Deployment of `MultiCallTest` (the last of 4 deploys) |
| **Contracts deployed** | `Simple`, `ERC20`, `ProgramTest`, `MultiCallTest` |
| **EVM opcodes exercised** | 4 constructor executions |
| **State changes** | 4 new contract accounts with code |
| **Expected block size** | Small (only last deploy is recorded) |

Deploys 4 different Solidity contracts in sequence. Each deployment is its own block.
The recorded block is the last one (`MultiCallTest`). The prior 3 deployments warm up
the state trie. Useful for benchmarking a deployment block after significant prior state
growth.

---

#### 8. `TestRecordBlockLargeContractDeploy`

| Attribute | Value |
|---|---|
| **Recorded block contains** | Deployment of a ~24 KB contract |
| **Contracts deployed** | Synthetic contract (24,000 bytes, near the EIP-170 limit) |
| **EVM opcodes exercised** | CODECOPY in initcode |
| **State changes** | 1 new contract account with ~24 KB of code |
| **Expected block size** | Large |

Deploys a contract with 24,000 bytes of bytecode (just under the 24,576-byte EIP-170
limit). The bytecode starts with `PUSH1 0 PUSH1 0 RETURN` and is padded with `STOP`
opcodes. This produces one of the largest possible single-transaction blocks in terms
of raw code storage. Good for benchmarking JIT handling of large state insertions.

---

### Category 3 — Solidity Contract Calls

These tests produce blocks where the recorded transaction calls an already-deployed
Solidity contract. They exercise the EVM interpreter with varying amounts of state
access.

#### 9. `TestRecordBlockSolidityRepeatedIncrements`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `Simple.Increment()` call (the 20th) |
| **EVM opcodes exercised** | SLOAD, ADD, SSTORE |
| **State changes** | 1 storage slot update (counter) |
| **Expected block size** | Tiny |

Deploys `Simple` and calls `Increment()` 20 times. Each call reads the counter,
increments it, and writes it back. Only the last call's block is recorded. The
interesting aspect is that the storage slot has been written 19 times already, so this
benchmarks JIT against a "warm" storage slot.

---

#### 10. `TestRecordBlockERC20Transfers`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `ERC20.transfer()` call (the 10th) |
| **EVM opcodes exercised** | SLOAD, SUB, ADD, SSTORE, LOG3 (Transfer event) |
| **State changes** | 2 balance slot updates + 1 event log |
| **Expected block size** | Small |

Deploys an ERC20 token, generates 10 recipient accounts, and transfers 1000 tokens to
each. The recorded block is the last transfer. An ERC20 `transfer()` modifies two
storage slots (sender balance, receiver balance) and emits a `Transfer` event with 3
indexed topics. This represents a typical DeFi-style operation.

---

#### 11. `TestRecordBlockPrecompileCalls`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `Simple.StoreDifficulty()` call |
| **EVM opcodes exercised** | DIFFICULTY, SSTORE |
| **State changes** | 1 storage slot (stores current block difficulty) |
| **Expected block size** | Tiny |

Reads `ArbSys.ArbBlockNumber` (off-chain call, no block produced), deploys `Simple`,
then calls `StoreDifficulty()` which executes the `DIFFICULTY` opcode and stores the
result. This tests JIT handling of Arbitrum-specific opcode overrides (DIFFICULTY
returns the ArbOS-defined value on Arbitrum).

---

#### 12. `TestRecordBlockERC20FullWorkflow`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `ERC20.transferFrom()` call |
| **EVM opcodes exercised** | SLOAD, SUB, ADD, SSTORE, LOG3, allowance checks |
| **State changes** | 3 storage slot updates (sender balance, receiver balance, allowance) + event |
| **Expected block size** | Small |

Full ERC20 lifecycle: deploy, fund 5 users with ETH, transfer tokens to each, set
approvals between users, then execute a `transferFrom`. The recorded block is the
`transferFrom` at the end. This is the most state-heavy ERC20 test — the trie has been
modified by ~15 prior transactions (ETH transfers, token transfers, approvals).
`transferFrom` reads/writes 3 storage slots (balance from, balance to, allowance).

---

### Category 4 — Stylus WASM Deployments

These tests produce blocks where Stylus (Arbitrum's WASM execution environment) programs
are deployed. WASM deployment blocks are larger than Solidity deployments because the
WASM bytecode is compressed and includes a Stylus prefix header.

#### 13. `TestRecordBlockWasmDeploy`

| Attribute | Value |
|---|---|
| **Recorded block contains** | Deployment of `storage.wasm` |
| **WASM program** | `storage` (reads/writes storage slots via Stylus hostio) |
| **State changes** | 1 new contract account with compressed WASM bytecode |
| **Expected block size** | Medium |

Deploys the `storage.wasm` Stylus program by sending compressed WASM bytecode as a
contract creation transaction. The bytecode goes through `readWasmFile` which compresses
it with `arbcompress` and prepends a Stylus version header. **Note**: this test does
*not* activate the program (unlike `deployWasm`), so the recorded block is purely the
raw deployment. Activation would be a separate transaction.

---

#### 14. `TestRecordBlockMultipleWasmDeploys`

| Attribute | Value |
|---|---|
| **Recorded block contains** | Activation of `math.wasm` (the last deploy/activate) |
| **WASM programs deployed** | `storage`, `keccak`, `multicall`, `math` |
| **State changes** | 4 contract creations + 4 ArbWasm activations |
| **Expected block size** | Medium |

Deploys and activates 4 different Stylus programs. Each `deployWasm` call produces 2
blocks (deploy + activate). The recorded block is the latest one (the activation of
`math.wasm`). This tests JIT against a chain state that has multiple compiled WASM
modules cached.

---

### Category 5 — Stylus WASM Execution

These tests produce blocks where the recorded transaction is a call into an
already-deployed and activated Stylus WASM program. The JIT must execute compiled WASM
code via Stylus hostio syscalls.

#### 15. `TestRecordBlockWasmStorageWrite`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus storage write |
| **WASM program called** | `storage` |
| **Hostio calls** | `storage_store_bytes32` |
| **State changes** | 1 storage slot written |
| **Expected block size** | Tiny |

The WASM equivalent of `TestRecordBlockSolidityRepeatedIncrements` but with a single
write. Deploys and activates `storage.wasm`, then calls it with `argsForStorageWrite`
to write a random key-value pair. The Stylus program reads the calldata, parses the
opcode byte (0x01 = write), and calls the `storage_store_bytes32` hostio. Minimal WASM
execution block.

---

#### 16. `TestRecordBlockWasmMultipleStorageWrites`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus storage write (the 20th) |
| **WASM program called** | `storage` |
| **Hostio calls** | `storage_store_bytes32` |
| **State changes** | 1 storage slot written (20 total across all blocks) |
| **Expected block size** | Tiny |

Same as above but repeats the write 20 times with random keys and values. Each write is
a separate transaction/block. The recorded block is the last one. Tests JIT performance
on a WASM call with 20 prior storage writes in the trie.

---

#### 17. `TestRecordBlockWasmMulticallStorageOps`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus multicall executing 16 CALL→storage writes |
| **WASM programs called** | `multicall` → `storage` (16 times) |
| **Hostio calls** | 16× `call_contract` + 16× `storage_store_bytes32` |
| **State changes** | 16 storage slots written |
| **Expected block size** | Medium |

Deploys `multicall.wasm` and `storage.wasm`. Constructs a single multicall transaction
that performs 16 sequential `CALL`s from multicall into storage, each writing a different
random key-value pair. All 16 cross-contract calls happen in **one transaction / one
block**. This is a significantly heavier block than single-write tests because the JIT
must handle many Stylus-to-Stylus cross-contract calls.

---

#### 18. `TestRecordBlockWasmKeccak`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `ProgramTest.CallKeccak()` → `keccak.wasm` |
| **WASM program called** | `keccak` (via Solidity `ProgramTest` proxy) |
| **Hostio calls** | `native_keccak256` |
| **State changes** | Minimal (proxy call overhead) |
| **Expected block size** | Small |

Deploys `keccak.wasm` and a Solidity `ProgramTest` proxy. Calls the proxy's
`CallKeccak` method which forwards to the WASM program. The WASM program hashes a
76-byte preimage using the `native_keccak256` hostio. This is **compute-heavy** — the
keccak operation dominates, with minimal storage I/O. Good for benchmarking JIT's
cryptographic hostio performance.

---

#### 19. `TestRecordBlockWasmMath`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `ProgramTest.MathTest()` → `math.wasm` |
| **WASM program called** | `math` (via Solidity `ProgramTest` proxy) |
| **Hostio calls** | Various math-related hostios |
| **State changes** | Minimal |
| **Expected block size** | Small–medium |

Deploys `math.wasm` and calls it through the `ProgramTest` Solidity proxy. The math
program exercises arbitrary-precision arithmetic operations via Stylus hostios
(addition, multiplication, modular exponentiation, etc.). This is the most
**compute-intensive** WASM test — minimal storage I/O, heavy ALU work. Ideal for
benchmarking raw JIT computation throughput.

---

#### 20. `TestRecordBlockWasmCreate`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus `CREATE` opcode execution |
| **WASM program called** | `create` |
| **Hostio calls** | `create1` |
| **State changes** | New contract account created from within WASM + value transfer |
| **Expected block size** | Medium–large |

Deploys `create.wasm`, then calls it with initcode for `storage.wasm`. The WASM program
executes a `CREATE` opcode, deploying a new contract from within a Stylus execution
context. The initcode wraps the compressed WASM of `storage.wasm`, making the calldata
quite large. A random ETH value is transferred to the newly created contract. This tests
the JIT's handling of the CREATE hostio — one of the most complex operations in Stylus.

---

#### 21. `TestRecordBlockWasmLogs`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus log emission (4 topics + 128 B data) |
| **WASM program called** | `log` |
| **Hostio calls** | `emit_log` |
| **State changes** | None (only log/event emission) |
| **Expected block size** | Tiny |

Deploys `log.wasm` and calls it with arguments encoding 4 topics (each 32 bytes) and
128 bytes of random data. The WASM program calls the `emit_log` hostio to emit a single
EVM log. No storage is modified. The block contains only the log receipt data. This
benchmarks JIT's log emission path.

---

#### 22. `TestRecordBlockWasmDeepMulticall`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus multicall, 8 levels deep |
| **WASM programs called** | `multicall` (7 nested calls to itself) → `storage` |
| **Hostio calls** | 8× `call_contract` + 1× `storage_store_bytes32` |
| **State changes** | 1 storage slot written |
| **Expected block size** | Small–medium |

Deploys `multicall.wasm` and `storage.wasm`. Constructs a deeply nested call chain:
`multicall → multicall → multicall → ... → storage write` (8 levels total). Each level
is a Stylus-to-Stylus cross-contract `CALL`. Only one storage slot is actually written
at the bottom of the call stack. This tests JIT performance under deep call-stack
pressure — useful for benchmarking context switching and stack management overhead.

---

#### 23. `TestRecordBlockWasmLargeMulticall`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus multicall with 64 operations (32 writes + 32 reads) |
| **WASM programs called** | `multicall` → `storage` (64 cross-contract calls) |
| **Hostio calls** | 64× `call_contract` + 32× `storage_store_bytes32` + 32× `storage_load_bytes32` |
| **State changes** | 32 storage slots written |
| **Expected block size** | Large |

The heaviest single-transaction block in the suite. Deploys `multicall.wasm` and
`storage.wasm`, then builds a single multicall that performs 32 write-then-read cycles.
Each cycle is 2 cross-contract `CALL`s (write + read), for a total of 64 calls in one
transaction. This produces a **large block** with heavy I/O — 32 `SSTORE`s, 32
`SLOAD`s, and 64 cross-contract calls.

---

#### 24. `TestRecordBlockWasmMulticallStoreAndLog`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus nested multicall with stores + loads + logs |
| **WASM programs called** | `multicall` (2 levels) |
| **Hostio calls** | 16× `storage_store_bytes32` + 8× `storage_load_bytes32` + 24× `emit_log` + 1× `call_contract` |
| **State changes** | 16 storage slots written + 24 event logs emitted |
| **Expected block size** | Medium–large |

Deploys `multicall.wasm` and `storage.wasm`. Builds inner multicall args using the
`multicallAppendStore` helper (which uses the multicall program's inline storage
operations with log emission enabled) and `multicallAppendLoad` (inline loads with log
emission). The inner args are then wrapped in an outer `CALL` to multicall. The result
is a single transaction that writes 16 storage slots, reads 8 storage slots, and emits
24 logs — a mixed I/O + logging workload.

---

#### 25. `TestRecordBlockWasmMultipleCreates`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus `CREATE` (the 3rd) |
| **WASM program called** | `create` |
| **Hostio calls** | `create1` |
| **State changes** | 1 new contract account + value transfer |
| **Expected block size** | Medium–large |

Deploys `create.wasm` and calls it 3 times, each time with `storage.wasm` initcode and
a random ETH value. The recorded block is the 3rd `CREATE`. This tests JIT handling of
`CREATE` after prior contract creation — the state trie has 2 additional contract
accounts from earlier blocks.

---

### Category 6 — Mixed Workloads

These tests combine multiple transaction types within a single test run. The recorded
block is always the last transaction, but the chain state reflects all prior operations.

#### 26. `TestRecordBlockMixedEthAndSolidity`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `Simple.IncrementEmit()` call |
| **Prior transactions** | ETH transfer, `Simple` deployment, `Simple.Increment()` |
| **EVM opcodes exercised** | SLOAD, ADD, SSTORE, LOG (counter event) |
| **State changes** | 1 storage slot + 1 event |
| **Expected block size** | Tiny |

Executes a mixed sequence: ETH transfer → deploy Simple → Increment → IncrementEmit.
The recorded block is the `IncrementEmit` call, which increments a counter and emits an
event. The interesting aspect is the mixed state context: the trie has a transfer, a
deployment, and a prior state mutation before the recorded block.

---

#### 27. `TestRecordBlockMixedSolidityAndWasm`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 Stylus storage write |
| **Prior transactions** | `Simple` deployment, `storage.wasm` deploy+activate, `Simple.Increment()` |
| **Hostio calls** | `storage_store_bytes32` |
| **State changes** | 1 storage slot written |
| **Expected block size** | Tiny |

Deploys both a Solidity contract (`Simple`) and a Stylus program (`storage.wasm`), calls
the Solidity contract, then calls the WASM program. The recorded block is the WASM
storage write. This tests JIT performance when the state trie contains both Solidity and
WASM contract deployments.

---

#### 28. `TestRecordBlockMixedAll`

| Attribute | Value |
|---|---|
| **Recorded block contains** | 1 `ProgramTest.CallKeccak()` → `keccak.wasm` |
| **Prior transactions** | 3 ETH transfers, `Simple` deploy, `storage.wasm` deploy+activate, `keccak.wasm` deploy+activate, `Simple.IncrementEmit()`, Stylus storage write, `ProgramTest` deploy |
| **Hostio calls** | `native_keccak256` |
| **State changes** | Minimal (keccak is pure compute) |
| **Expected block size** | Small |

The "kitchen sink" test. Executes every type of transaction before recording: ETH
transfers, Solidity deployment, Stylus deployments, Solidity state-mutating call,
Stylus storage write, then finally a Stylus keccak computation. The recorded block is
the keccak call at the end. The trie at this point has the most diverse state of any
test — multiple EOA balance changes, Solidity code, WASM code, activated WASM modules,
and storage writes from both Solidity and Stylus. This tests JIT under the most
realistic complex-state conditions.

---

## Summary Table

| Test | Recorded Tx Type | Block Size | Key Characteristic |
|---|---|---|---|
| `SingleTransfer` | ETH transfer | Tiny | Baseline — minimal block |
| `MultipleTransfers` | ETH transfer | Tiny | After 4 prior transfers |
| `ManyTransfers` | ETH transfer | Tiny | After 49 prior transfers |
| `TransfersWithCalldata` | ETH transfer + 4 KB data | Small | Large tx payload |
| `SolidityDeploy` | Contract creation + call | Small | Simple contract |
| `ERC20Deploy` | Contract creation | Medium | Larger contract bytecode |
| `MultipleSolidityDeploys` | Contract creation | Small | 4th deploy in sequence |
| `LargeContractDeploy` | Contract creation | Large | 24 KB near EIP-170 limit |
| `SolidityRepeatedIncrements` | Solidity SSTORE | Tiny | 20th write to same slot |
| `ERC20Transfers` | ERC20 transfer | Small | Token balance + event |
| `PrecompileCalls` | Solidity DIFFICULTY+SSTORE | Tiny | Arb precompile override |
| `ERC20FullWorkflow` | ERC20 transferFrom | Small | Allowance + 2 balances |
| `WasmDeploy` | WASM deployment | Medium | Compressed Stylus bytecode |
| `MultipleWasmDeploys` | WASM activation | Medium | 4th WASM activate |
| `WasmStorageWrite` | Stylus storage write | Tiny | 1 hostio call |
| `WasmMultipleStorageWrites` | Stylus storage write | Tiny | 20th write |
| `WasmMulticallStorageOps` | Stylus multicall (16 writes) | Medium | 16 cross-contract calls |
| `WasmKeccak` | Stylus keccak hash | Small | Compute-heavy |
| `WasmMath` | Stylus math ops | Small–medium | Most compute-intensive |
| `WasmCreate` | Stylus CREATE | Medium–large | Contract from WASM |
| `WasmLogs` | Stylus log emission | Tiny | 4 topics + 128 B |
| `WasmDeepMulticall` | Stylus nested calls (8 deep) | Small–medium | Call-stack stress |
| `WasmLargeMulticall` | Stylus multicall (64 ops) | Large | Heaviest single tx |
| `WasmMulticallStoreAndLog` | Stylus stores + loads + logs | Medium–large | Mixed I/O + logging |
| `WasmMultipleCreates` | Stylus CREATE (3rd) | Medium–large | Repeated contract creation |
| `MixedEthAndSolidity` | Solidity call + event | Tiny | Mixed Solidity context |
| `MixedSolidityAndWasm` | Stylus storage write | Tiny | Mixed Solidity + WASM state |
| `MixedAll` | Stylus keccak | Small | Diverse state — all tx types |


Failed tests:
  - TestRecordBlockERC20FullWorkflow
