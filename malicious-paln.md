## Executive summary

We implemented an **experimental “malicious node” mode** in Nitro to deliberately create an execution divergence between a malicious validator and an honest validator, while still allowing the malicious validator to participate in the full **BOLD** challenge flow all the way to **One-Step Proof (OSP)**.

The core idea is:

- Make the malicious node compute a **deterministically wrong** result by changing the semantics of one **Arbitrum-specific Host I/O opcode**: `ReadInboxMessage`.
- Ensure the same “wrong” behavior is applied consistently across **all local execution environments** used by the node (execution layer, JIT, arbitrator).
- Make sure the node believe itself the honest validator all the time until the challenge failed, adding wasm module root cheating mechanism
- Work around practical issues required to reach OSP (module-root / replay binary selection, stage transitions, and transaction preflight / gas estimation).

This document summarizes what we changed and why.

## Goal and constraints

### Goal

Run a fraud proof experiment where:

1. An honest node and a malicious node disagree on execution.
2. The BOLD protocol narrows the disagreement through block → large steps → small steps.
3. The dispute reaches **One-Step Proof (OSP)** on L1.
4. The malicious node is treated as “honest” locally (i.e., it keeps acting/participating) and only “realizes” it loses when the on-chain OSP uses the correct semantics.

### Key constraints we had to satisfy

- **Divergence must be Arbitrum-specific**, not a generic WASM opcode tweak, to reduce unintended side effects.
- **Execution environments must match**: execution layer (Go), JIT, and arbitrator must compute the same (wrong) result on the malicious node, otherwise the node self-detects inconsistency and cannot proceed cleanly.
- **Replay binary selection is module-root keyed** in multiple places. When the chain’s on-chain WASM module root differs from local build artifacts, the malicious node must still load a usable local machine (e.g., `latest`) and must not get stuck when the challenge transitions stages and requests proof data/hashes. (Especially when enter to new edge)
- **OSP transaction flow**: at the final step, preflight checks (including `eth_estimateGas`) may revert due to invalid proofs from the malicious node’s perspective; we must still be able to submit the transaction so L1 can adjudicate.

## High-level approach

### Why `ReadInboxMessage`

We selected `ReadInboxMessage` (Host I/O) as the divergence point because:

- It is **Arbitrum-specific** (not a generic WASM operator), and it is already implemented separately in the L1 OSP prover.
- It is invoked during normal execution in many scenarios, so we can reach OSP reliably.
- The L1 OSP contract (`OneStepProverHostIo`) validates inbox message proofs on-chain, so the “correct behavior” is well-defined and does not depend on the malicious node’s local changes.

### What the malicious behavior is

We introduced a deterministic mutation in the sequencer batch bytes returned via `ReadInboxMessage`:

- Flip a single bit at a fixed byte offset (`offset = 64`, mask `0x01`) in the serialized batch message.
- Apply the mutation consistently so the malicious node’s execution is self-consistent, but differs from the honest node and from the on-chain OSP semantics.

We deliberately avoid mutating the first 40 bytes of the batch header to reduce the chance of breaking structural parsing early.

## Implementation details (by component)

### 1) Malicious-mode configuration plumbing

**Files**

- `validator/valnode/valnode.go`
- `util/malicious/config.go`
- `cmd/nitro/nitro.go`
- `cmd/nitro-val/nitro_val.go`

**What changed**

- Added config flags under `validation.wasm.*`:
    - `malicious-mode`
    - `override-module-root`
    - `allow-gas-estimation-failure`
- Introduced a tiny in-process config singleton (`util/malicious/config.go`) that exposes helpers:
    - `Enabled()`
    - `OverrideWasmModuleRoot()`
    - `AllowGasEstimationFailure()`
    - `MutateInboxMessage(...)`
- When `malicious-mode` is enabled, we also set an env var:
    - `NITRO_MALICIOUS_MODE=1`

**Why**

- We need a uniform switch that can be read in Go code paths (execution/challenge) and also in Rust code paths (JIT / arbitrator), hence the env var.
- Keeping these knobs in `validation.wasm.*` ensures the malicious behavior is scoped to the validator use-cases and can be toggled per node.

### 2) Deterministic divergence in `ReadInboxMessage`

**Files**

- Go execution input (for arbitrator/JIT inbox backend):
    - `arbnode/inbox_tracker.go`
- JIT host I/O:
    - `arbitrator/jit/src/wavmio.rs`
    - `arbitrator/jit/Cargo.toml` (dependency)
- Arbitrator (prover interpreter):
    - `arbitrator/prover/src/machine.rs`

**What changed**

- **Go side**: mutate sequencer batch bytes returned by the inbox multiplexer backend:
    - `multiplexerBackend.PeekSequencerInbox()` returns mutated bytes when malicious mode is enabled.
    - Important: we mutate the returned slice (copy), not the cached original, to avoid contaminating persisted batch data.
- **JIT**: in `read_inbox_message`, if the 32-byte chunk covers the mutation offset, we flip the bit in that chunk before writing to guest memory.
- **Arbitrator**: same logic in the `Opcode::ReadInboxMessage` execution path.
- Both Rust paths are gated by `NITRO_MALICIOUS_MODE`.

**Why**

- This is the core divergence mechanism.
- The chunk-aware mutation (relative index inside the 32-byte window) ensures correctness for host I/O that reads in 32-byte segments.
- By mutating the inbox data, we produce downstream differences in execution state, while L1 OSP still uses the correct data/proofs.

### 3) Keeping the malicious node “self-consistent”

**Files**

- `execution/gethexec/block_recorder.go`
- `staker/stateless_block_validator.go`
- `arbnode/transaction_streamer.go`

**What changed**

When malicious mode is enabled, several “self-consistency” checks that would otherwise stop/abort are downgraded to warnings:

- Canonical blockhash mismatch checks during recording.
- Validation recording mismatch checks.
- Consensus-vs-execution blockhash mismatch checks.

**Why**

- With a deliberate divergence, some internal invariants will not hold if compared against unmodified expectations.
- We need the malicious node to keep operating and participating in BOLD rather than shutting down early.

### 4) Forcing the validator to use the local `latest` machine (module root override)

**Files**

- `validator/server_common/malicious.go`
- `validator/server_arb/validator_spawner.go`
- `validator/server_jit/spawner.go`
- `staker/bold/bold_state_provider.go`
- `cmd/nitro/nitro.go`

**What changed**

- Added a helper `ResolveModuleRoot(locator, moduleRoot)` that, when `override-module-root` is enabled, maps any requested module root to `locator.LatestWasmModuleRoot()`.
- Applied this mapping:
    - In arbitrator spawner and JIT spawner (before loading machines).
    - In BOLD state provider for:
        - Challenge cache keying (`WavmModuleRoot`)
        - Machine hash collection calls
        - Proof collection calls
- Skipped `checkWasmModuleRootCompatibility` on startup when malicious override is enabled.

**Why**

- In local experiments, the chain’s on-chain wasm module root may not match the locally built machine artifacts.
- Without overriding, the node would look up replay binaries by the on-chain root and fail to load/prove.
- BOLD also namespaces cache entries by module root; we must ensure cache keys and machine loading refer to the same effective root to avoid cache misses and stage-transition failures.

### 5) Allowing OSP transaction submission even if estimation/preflight reverts

**Files**

- `bold/chain-abstraction/sol-implementation/transact.go`

**What changed**

- When `allow-gas-estimation-failure` is enabled:
    - We set a fixed fallback `GasLimit = 5_000_000`.
    - We also set `opts.GasLimit` **before** calling the “NoSend preflight” `fn(opts)` to avoid abigen calling `EstimateGas` internally.
    - If the preflight still errors, we retry once with the fallback gas limit.
    - If the external `backend.EstimateGas` fails, we use the same fallback gas.

**Why**

- The BOLD code previously did a “test execution” by setting `opts.NoSend=true` and calling `fn(opts)`; if `GasLimit==0`, abigen tries to estimate gas, which can revert with errors like “Invalid inclusion proof”.
- In this experiment, that revert is expected (the malicious node provides a proof inconsistent with L1 semantics), but we still need to **send the transaction** so L1 can adjudicate and finalize the challenge.

## How to run a malicious node

On the malicious node only:

- `-validation.wasm.malicious-mode=true`
- `-validation.wasm.override-module-root=true`
- `-validation.wasm.allow-gas-estimation-failure=true`

Note:

- You still need a machine directory containing `machine.wavm.br` and `replay.wasm`. The `nitro-node-dev` docker target includes `replay.wasm` under `/home/user/target/machines/latest/`.
- Honest nodes must run without these flags.

## Build/verification performed

- `cargo check -p jit -p prover` under `arbitrator/` (successful)
- Go build verification via `go test -run TestNonExistent` across key packages (successful)

## Known limitations / follow-ups

- The mutation offset is currently a fixed constant (`64`). If a particular workload’s batches are shorter, or if the payload format changes, the mutation may not take effect and divergence may not be triggered. Making the mutation offset configurable (or keyed to message type) is a possible follow-up.
- `override-module-root` is intentionally unsafe and should never be used outside controlled experiments.
- The approach assumes the divergence is sufficient to drive BOLD to OSP under the chosen scenario; depending on workload, you may want to tune the mutation to guarantee it impacts the relevant machine state path.