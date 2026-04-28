# Plan: BOLD Changes to Support MEL Assertion Determinism

## Context

PR [OffchainLabs/nitro-contracts#427](https://github.com/OffchainLabs/nitro-contracts/pull/427) changes the assertion determinism rule in the rollup contracts from **batch-count-based** (`InboxMaxCount`) to **parent-chain-block-hash-based** (`nextParentChainBlockHash`). This is the core of MEL's assertion model: instead of "process up to batch N", assertions now say "extract messages up to parent chain block with hash X".

The entire BOLD stack — assertion posting, state provider, challenge system, one-step proofs — currently threads `InboxMaxCount` (a batch count `*big.Int`) everywhere. All of these paths must be rewired to use `NextParentChainBlockHash` and derive batch/message counts from the MEL system.

**Current flow:**
```
Parent assertion → InboxMaxCount → batch count → message count → execution state
```
**New flow:**
```
Parent assertion → NextParentChainBlockHash → MEL state at that block → (BatchCount, MsgCount) → execution state
```

---

## Phase 0: Prerequisite — Regenerate Solidity Bindings

The generated Go bindings in `solgen/go/rollupgen/rollupgen.go` are stale relative to PR #427. They must be regenerated after pointing the `contracts` submodule to the PR 427 commit.

**Steps:**
1. Update `contracts` submodule to the PR 427 branch
2. Run `make contracts` (or `make all` which depends on `.make/solgen` → `go run solgen/gen.go`)

**Key binding changes expected:**
- `GlobalState.Bytes32Vals`: `[2][32]byte` → `[4][32]byte` (adds MELStateHash, MELMsgHash)
- `ConfigData.NextInboxPosition uint64` → `ConfigData.NextParentChainBlockHash [32]byte`
- `AssertionInputs`: gains `AfterMELState MELState` field
- `AssertionCreated` event: drops `AfterInboxBatchAcc`/`InboxMaxCount`, gains `NextParentChainBlockHash`
- New `MELState` struct type in bindings

**Files affected (auto-generated):**
- [rollupgen.go](solgen/go/rollupgen/rollupgen.go)
- [challengeV2gen.go](solgen/go/challengeV2gen/challengeV2gen.go)

---

## Phase 1: Core Data Structure Changes

### 1.1 Add `NextParentChainBlockHash` to `AssertionCreatedInfo` (keep old fields)

**File:** [interfaces.go](bold/protocol/interfaces.go) (lines 115-129)

**Keep** `InboxMaxCount` and `AfterInboxBatchAcc` for pre-MEL backwards compatibility. **Add** `NextParentChainBlockHash` for post-MEL. A nitro node handles both pre-MEL assertions (using `InboxMaxCount`) and post-MEL assertions (using `NextParentChainBlockHash`). Downstream code checks which field to use based on MEL activation status.

```go
type AssertionCreatedInfo struct {
    ...
    InboxMaxCount            *big.Int    // Pre-MEL determinism (zero post-MEL)
    AfterInboxBatchAcc       common.Hash // Pre-MEL (zero post-MEL)
    NextParentChainBlockHash common.Hash // Post-MEL determinism (zero pre-MEL)
    ...
}
```

### 1.2 Update `GoGlobalState.Hash()` — backwards-compatible MEL field hashing

The Rust prover ([machine.rs](crates/prover/src/machine.rs) lines 840-881) uses a **backwards-compatible** hashing strategy: it only hashes `bytes32` values up to the last non-zero index, with a minimum of index 1. This means:

- **Pre-MEL** (MELStateHash=0, MELMsgHash=0): hashes `[BlockHash, SendRoot]` + u64s → **same hash as today**
- **Post-MEL** (any MEL field non-zero): hashes `[BlockHash, SendRoot, MELStateHash, MELMsgHash]` + u64s

The Go `Hash()` must replicate this `bytes32_last_non_zero_index()` logic:

**File:** [execution_state.go](bold/protocol/execution_state.go) (lines 49-56)
```go
func (s GoGlobalState) Hash() common.Hash {
    data := []byte("Global state:")
    data = append(data, s.BlockHash.Bytes()...)
    data = append(data, s.SendRoot.Bytes()...)
    // Only include MEL fields if they are non-zero (backwards compatibility)
    if s.MELMsgHash != (common.Hash{}) {
        data = append(data, s.MELStateHash.Bytes()...)
        data = append(data, s.MELMsgHash.Bytes()...)
    } else if s.MELStateHash != (common.Hash{}) {
        data = append(data, s.MELStateHash.Bytes()...)
    }
    data = append(data, u64ToBe(s.Batch)...)
    data = append(data, u64ToBe(s.PosInBatch)...)
    return crypto.Keccak256Hash(data)
}
```

The Solidity `GlobalState.hash()` in PR #427 currently hashes all 4 values unconditionally — **this needs to be updated in the contracts PR to match the Rust prover's backwards-compatible approach**, or the Rust approach needs to change. This is a cross-repo alignment issue to resolve.

**File:** [execution_state.go](validator/execution_state.go) (lines 53-59)
- Same change — both `Hash()` implementations must be identical

> **Not consensus-breaking for pre-MEL states:** When MEL fields are zero, the hash output is identical to today's. Only post-MEL states (with non-zero MEL fields) get the new hash format.

### 1.3 Update `GoGlobalStateFromSolidity` and `AsSolidityStruct`

**File:** [execution_state.go](bold/protocol/execution_state.go) (lines 28-35, 58-63)

`GoGlobalStateFromSolidity`: map `Bytes32Vals[2]` → `MELStateHash`, `Bytes32Vals[3]` → `MELMsgHash`

`AsSolidityStruct`: use `[4][32]byte{BlockHash, SendRoot, MELStateHash, MELMsgHash}`

### 1.4 Add `NextParentChainBlockHash` to `ConfigSnapshot` (keep old field)

**File:** [provider.go](bold/state/provider.go) (lines 38-44)

Keep `InboxMaxCount` for pre-MEL, add `NextParentChainBlockHash` for post-MEL.

---

## Phase 2: MEL Integration Layer

### 2.1 Define a MEL state lookup interface

The BOLD system needs to translate `NextParentChainBlockHash` → `(BatchCount, MsgCount)`. We need a thin interface to avoid coupling `bold/` packages to `arbnode/mel/`.

**Critical constraint:** The lookup **must only use validated MEL states** — states that have been confirmed by the `MELValidator` via the unified replay binary. Using unvalidated MEL states would let the BOLD staker post assertions based on potentially incorrect extraction, which defeats the purpose of validation.

**New interface** (add to `bold/state/provider.go` or a new file like `bold/state/mel.go`):

```go
type ValidatedMELStateLookup interface {
    // GetValidatedMELStateByBlockHash returns MEL state info at the given parent chain 
    // block hash, but ONLY if the extraction up to that block has been validated.
    // Returns ErrChainCatchingUp if MEL validation hasn't reached this block yet.
    GetValidatedMELStateByBlockHash(ctx context.Context, blockHash common.Hash) (*MELStateInfo, error)
}

type MELStateInfo struct {
    BatchCount             uint64
    MsgCount               uint64
    ParentChainBlockNumber uint64
    StateHash              common.Hash
}
```

### 2.2 Implement the interface

**New file or addition to:** [bold_state_provider.go](staker/bold/bold_state_provider.go)

The implementation wraps **both** `MELValidatorInterface` and `melrunner.MessageExtractor`:

1. Resolve `blockHash` → block number (via parent chain client `HeaderByHash`)
2. Check that `blockNumber <= melValidator.LatestValidatedParentChainBlock()` — **only use validated states**
3. Call `messageExtractor.GetState(blockNumber)` → `mel.State`
4. Return `{BatchCount, MsgCount, ParentChainBlockNumber, StateHash}`

If MEL validation hasn't reached the required block, return `ErrChainCatchingUp`. This mirrors the existing pattern where `isStateValidatedAndMessageCountPastThreshold` checks block validation status before posting assertions.

The key distinction from the current MEL validator interface:
- `MELValidatorInterface.LatestValidatedMELState()` only returns the latest validated state
- We need a method to get the validated MEL state **at a specific parent chain block hash**
- This may require adding a `GetValidatedMELStateAtBlock(blockNum uint64)` method to `MELValidatorInterface`, which checks `blockNum <= latestValidatedParentChainBlock` and then delegates to `messageExtractor.GetState(blockNum)`

### 2.3 Inject MEL dependency into assertion Manager and challenge Manager

- [manager.go](bold/assertions/manager.go): `Manager` struct gains `melLookup ValidatedMELStateLookup` field
- [challenges.go](bold/challenge/challenges.go): `Manager` struct gains `melLookup ValidatedMELStateLookup` field
- [tree.go](bold/challenge/tree/) challenge tree construction needs access via its metadata reader or a new field
- [bold_staker.go](staker/bold/bold_staker.go): Wire the lookup implementation into the assertion manager and challenge manager during BOLD staker initialization (around line 599)

---

## Phase 3: Assertion Posting Flow

### 3.1 Event parsing — read `nextParentChainBlockHash`

**File:** [bold_assertioncreation.go](staker/bold_assertioncreation.go) (lines 90-107)
- After binding regen, parse `parsedLog.NextParentChainBlockHash` instead of the removed fields
- Set `NextParentChainBlockHash` on the returned `AssertionCreatedInfo`

**File:** [assertion_chain.go](bold/protocol/sol/assertion_chain.go) (lines 1075-1092)
- Same change in the `AssertionChain.ReadAssertionCreationInfo` method

### 3.2 Assertion poster — use validated MEL to derive batch count

**File:** [poster.go](bold/assertions/poster.go) (lines 178-251)

Current (line 189-194):
```go
batchCount := parentCreationInfo.InboxMaxCount.Uint64()
```

New:
```go
melInfo, err := m.melLookup.GetValidatedMELStateByBlockHash(ctx, parentCreationInfo.NextParentChainBlockHash)
if err != nil {
    if errors.Is(err, state.ErrChainCatchingUp) {
        // MEL hasn't validated up to this block yet — wait and retry
        return none, nil
    }
    return none, err
}
batchCount := melInfo.BatchCount
```

The overflow assertion check at line 216 (`newState.GlobalState.Batch < batchCount`) stays conceptually the same but uses MEL-derived `batchCount`.

### 3.3 `ExecutionStateAfterParent` — thread validated MEL batch count

**File:** [manager.go](bold/assertions/manager.go) (lines 384-387)

Change from:
```go
return m.execProvider.ExecutionStateAfterPreviousState(ctx, parentInfo.InboxMaxCount.Uint64(), goGlobalState)
```
To:
```go
melInfo, err := m.melLookup.GetValidatedMELStateByBlockHash(ctx, parentInfo.NextParentChainBlockHash)
if err != nil { return nil, err }
return m.execProvider.ExecutionStateAfterPreviousState(ctx, melInfo.BatchCount, goGlobalState)
```

This ensures we **never post an assertion based on unvalidated MEL extraction**. If MEL validation is behind, `ErrChainCatchingUp` propagates up and the poster waits.

### 3.4 `ExecutionProvider` interface signature

**File:** [provider.go](bold/state/provider.go) (lines 61-68)

The signature `ExecutionStateAfterPreviousState(ctx, maxInboxCount uint64, previousGlobalState)` can remain unchanged — the parameter just gets its value from MEL instead of `InboxMaxCount`. The `uint64` represents the batch count in both cases.

### 3.5 Assertion hash computation — remove inboxAcc

**File:** [assertion_chain.go](bold/protocol/sol/assertion_chain.go) (lines 597-702)

- Line 603: Remove `InboxMaxCount.IsUint64()` check
- Line 624: `inboxBatchAcc` is already unused (comment says `PR 427: inboxAcc no longer used`)
- Line 625: `ComputeAssertionHash` already takes only `parentAssertionHash` and `afterState`
- Lines 654-670: Update `AssertionInputs` to include `AfterMELState` and `ConfigData.NextParentChainBlockHash`
- Line 658: `SequencerBatchAcc` field in `BeforeStateData` — review whether still needed

### 3.6 Assertion confirmation

**File:** [assertion_chain.go](bold/protocol/sol/assertion_chain.go) (lines 830-872)

- Line 845: Remove `prevCreationInfo.InboxMaxCount.IsUint64()` check
- Lines 857-861: `ConfigData` must use `NextParentChainBlockHash` from `prevCreationInfo` instead of `NextInboxPosition`

**File:** [assertion_chain.go](bold/protocol/sol/assertion_chain.go) (lines 874-898) — FastConfirmAssertion
- Already updated to drop `inboxAcc` arg, no further changes needed

---

## Phase 4: Challenge System Changes

All places that set `BatchLimit` from `InboxMaxCount` must derive it from MEL instead.

### 4.1 Challenge initiation — `addBlockChallengeLevelZeroEdge`

**File:** [challenges.go](bold/challenge/challenges.go) (lines 128-138)

Change:
```go
BatchLimit: state.Batch(prevCreationInfo.InboxMaxCount.Uint64()),
```
To MEL lookup:
```go
melInfo, err := m.melLookup.GetMELStateByBlockHash(ctx, prevCreationInfo.NextParentChainBlockHash)
// ...
BatchLimit: state.Batch(melInfo.BatchCount),
```

### 4.2 Edge tracker metadata — challenge manager

**File:** [manager.go](bold/challenge/manager.go) (lines 238-247)

Same pattern — replace `prevCreationInfo.InboxMaxCount.Uint64()` with MEL-derived `BatchCount`.

### 4.3 Edge history commitment preparation

**File:** [add_edge.go](bold/challenge/tree/add_edge.go) (lines 137-147)

Same pattern — replace `parentCreationInfo.InboxMaxCount.Uint64()` with MEL-derived `BatchCount`.

### 4.4 One-step proof execution context

**File:** [edge_challenge_manager.go](bold/protocol/sol/edge_challenge_manager.go) (lines 772, 799)

- Line 772: Remove `creationInfo.InboxMaxCount.IsUint64()` check
- Line 799: `MaxInboxMessagesRead` — per PR #427, this is set to `type(uint256).max` as a stopgap. The OSP contract changes are pending, so this may stay as-is for now.

### 4.5 Rival assertion handling

**File:** [sync.go](bold/assertions/sync.go) (lines 386-392)

Replace `args.invalidAssertion.InboxMaxCount` with `NextParentChainBlockHash` for logging and MEL-derived batch count for the execution state computation.

---

## Phase 5: Validation Pipeline — How MEL Validation Feeds Into BOLD

This section covers the critical path from MEL validation → block validation → BOLD state provider, which is what makes the BOLD staker work post-MEL.

### 5.0 Background: The Validation Pipeline Architecture

The validation pipeline is a **two-stage** process post-MEL:

**Stage 1 — MEL Validation** ([mel_validator.go](staker/mel_validator.go)):
1. `MELValidator.CreateNextValidationEntry()` walks parent chain blocks from `lastValidatedParentChainBlock + 1` upward
2. For each block, it runs `melextraction.ExtractMessages()` in **recording mode** — recording all preimages (block headers, transactions, receipts, blob data) needed to replay extraction deterministically
3. It verifies the recorded extraction matches the native extraction (`endState.Hash() != wantState.Hash()`)
4. Per-message preimages are cached in `msgPreimagesAndStateCache[msgIndex]` → `{msgPreimages, relevantState}` for later use by block validation
5. The validation entry's `Start`/`End` use `GoGlobalState` with `MELStateHash` set and `MELMsgHash = common.Hash{}` (indicating MEL extraction turn)
6. The unified replay binary (`cmd/unified-replay/main.go`) runs the same extraction via `extractMessagesUpTo()` using WAVM preimage resolution

**Stage 2 — Block Validation** ([block_validator.go](staker/block_validator.go) + [block_validation_entry_creator.go](staker/block_validation_entry_creator.go)):
1. `BlockValidator.createNextValidationEntry()` checks `config.EnableMEL` — if true, uses `MELEnabledValidationEntryCreator`
2. `MELEnabledValidationEntryCreator.CreateBlockValidationEntry()`:
   - Checks `latestValidatedMELState.MsgCount > position` (MEL must have validated extraction of this message first)
   - Fetches per-message preimages from `melValidator.FetchMsgPreimagesAndRelevantState(position)`
   - Constructs `startGS` with `{BlockHash, SendRoot, MELStateHash: relevantMELState.Hash(), MELMsgHash: msg.Hash(), PosInBatch: position}`
   - Constructs `endGS` similarly with execution result and next message hash
3. The unified replay binary's `produceBlock()` function runs the message through `arbos.ProduceBlock()` to validate the L2 block

**Key interaction with BOLD state provider:**
- `BOLDStateProvider.isStateValidatedAndMessageCountPastThreshold()` checks `BlockValidator.ReadLastValidatedInfo()` — this returns the `lastValidGS` which post-MEL includes `MELStateHash`/`MELMsgHash`
- The block validator's `updatelastValidGSIfNeeded()` detects when `MELMsgHash == common.Hash{}` (MEL extraction boundary) and enriches the state with MEL fields from `melValidator.FetchMessageOriginMELStateHash()`

### 5.1 `BOLDStateProvider` needs MEL access for GlobalState construction

**File:** [bold_state_provider.go](staker/bold/bold_state_provider.go)

The `BOLDStateProvider` struct needs a new dependency: either `MELValidatorInterface` or a simplified interface. This is needed because:

1. **`findGlobalStateFromMessageCountAndBatch`** (lines 317-351): Currently returns `GoGlobalState` with only `BlockHash`, `SendRoot`, `Batch`, `PosInBatch`. Must now also populate `MELStateHash` and `MELMsgHash`. Use `melValidator.FetchMessageOriginMELStateHash(count)` to get the MEL state hash, and construct the appropriate next message hash.

2. **`StatesInBatchRange`** (lines 237-311): Each `GoGlobalState` at lines 282-287 needs MEL fields for challenge history commitments. Each leaf state in the history commitment must include the correct `MELStateHash` and `MELMsgHash`, since these are now part of the `GoGlobalState.Hash()` used in `machineHash()`.

3. **`machineHash`** (line 313-315): `crypto.Keccak256Hash([]byte("Machine finished:"), gs.Hash().Bytes())` — since `gs.Hash()` now includes MEL fields, the machine hashes will automatically change. This is correct but means ALL history commitments change shape.

Add to `BOLDStateProvider` struct:
```go
type BOLDStateProvider struct {
    ...
    melValidator MELValidatorInterface  // NEW
}
```

### 5.2 `isStateValidatedAndMessageCountPastThreshold` — Dual validation check

**File:** [bold_state_provider.go](staker/bold/bold_state_provider.go) (lines 209-235)

Currently validates using `gs.Batch` comparisons against `lastValidatedGs.GlobalState.Batch`. Post-MEL:
- The `lastValidatedGs` from `BlockValidator.ReadLastValidatedInfo()` already contains MEL fields when `EnableMEL` is true
- Need to also verify that MEL extraction has been validated up to the required message count
- Can use `melValidator.LatestValidatedMELState()` and compare against the required MEL state

### 5.3 Unified replay binary — Global state for BOLD proofs

**File:** [unified-replay/main.go](cmd/unified-replay/main.go)

The replay binary is the **provable** execution that BOLD challenges verify via one-step proofs. Its flow is:

```
1. Read melMsgHash = melwavmio.GetMELMsgHash()
2. Read startMELState from preimage of melwavmio.GetStartMELRoot()

3. IF melMsgHash != 0x0 → produceBlock(melMsgHash)
   - This is executing a single L2 message into a block
   - Sets lastBlockHash, sendRoot via melwavmio

4. ELSE → extractMessagesUpTo(chainConfig, melState, targetBlockHash)
   - This is MEL extraction — walking backwards from targetBlockHash to melState.ParentChainBlockHash
   - Runs melextraction.ExtractMessages() for each block header
   - Sets endMELRoot via melwavmio

5. Check if more messages to execute: melState.MsgCount > positionInMEL
   - If yes: read next message, set MELMsgHash for next step
   - If no: set MELMsgHash = 0x0 (triggering extraction on next step)
```

The `targetBlockHash` comes from `melwavmio.GetEndParentChainBlockHash()` — this is the `nextParentChainBlockHash` from the assertion being proven. This is the OSP opcode that reads from the assertion's config data.

**Impact on BOLD**: When `CollectMachineHashes` or `CollectProof` are called for challenges, the validation entry's `StartState` and the machine's initial state must include MEL fields. The `CreateReadyValidationEntry` (used in [bold_state_provider.go](staker/bold/bold_state_provider.go) line 427) must produce entries with correct MEL state.

### 5.4 `CreateReadyValidationEntry` — Must include MEL preimages

**File:** [stateless_block_validator.go](staker/stateless_block_validator.go)

When BOLD's `CollectMachineHashes` or `CollectProof` calls `s.statelessValidator.CreateReadyValidationEntry(ctx, messageNum)`, the validation entry must include:
- MEL state preimages (the RLP-encoded MEL state whose hash is in `GoGlobalState.MELStateHash`)
- Message preimages from `melValidator.FetchMsgPreimagesAndRelevantState(messageNum)`
- The `StartState` GoGlobalState must include `MELStateHash` and `MELMsgHash`

This means `StatelessBlockValidator.CreateReadyValidationEntry()` needs MEL awareness — either by accepting MEL preimages as parameters, or by having its own MEL validator reference.

### 5.5 `ValidationInput.EndParentChainBlockHash` field

**File:** [validation_entry.go](validator/validation_entry.go)

The `ValidationInput` struct already has `EndParentChainBlockHash common.Hash`. This is used by the unified replay binary via `melwavmio.GetEndParentChainBlockHash()` to know how far to extract. For BOLD proofs, this must be set to the assertion's `nextParentChainBlockHash`.

---

## Phase 6: State Provider — Populate MEL Fields in GlobalState

### 6.1 `findGlobalStateFromMessageCountAndBatch`

**File:** [bold_state_provider.go](staker/bold/bold_state_provider.go) (lines 317-351)

Currently returns:
```go
return validator.GoGlobalState{
    BlockHash:  res.BlockHash,
    SendRoot:   res.SendRoot,
    Batch:      uint64(batchIndex),
    PosInBatch: uint64(count - prevBatchMsgCount),
}, nil
```

Must become:
```go
melStateHash, err := s.melValidator.FetchMessageOriginMELStateHash(count)
// ...
return validator.GoGlobalState{
    BlockHash:    res.BlockHash,
    SendRoot:     res.SendRoot,
    MELStateHash: melStateHash,
    MELMsgHash:   nextMsgHash, // hash of message at count, or 0x0 if at extraction boundary
    Batch:        uint64(batchIndex),
    PosInBatch:   uint64(count - prevBatchMsgCount),
}, nil
```

### 6.2 `StatesInBatchRange`

**File:** [bold_state_provider.go](staker/bold/bold_state_provider.go) (lines 237-311)

Each state at line 282-287 needs MEL fields:
```go
state := validator.GoGlobalState{
    BlockHash:    executionResult.BlockHash,
    SendRoot:     executionResult.SendRoot,
    MELStateHash: melStateHashForPos,  // NEW
    MELMsgHash:   melMsgHashForPos,    // NEW
    Batch:        batchNum,
    PosInBatch:   posInBatch,
}
```

This loop iterates over potentially many positions. For efficiency, batch MEL state lookups or cache the MEL state hash per parent chain block (since multiple messages may map to the same MEL state).

### 6.3 `virtualState` — Include MEL fields

**File:** [bold_state_provider.go](staker/bold/bold_state_provider.go) (lines 491-513)

The virtual state at line 505 also needs MEL fields:
```go
gs = option.Some(validator.GoGlobalState{
    BlockHash:    result.BlockHash,
    SendRoot:     result.SendRoot,
    MELStateHash: melStateHash,  // NEW
    MELMsgHash:   common.Hash{}, // virtual = at extraction boundary
    Batch:        uint64(limit),
    PosInBatch:   0,
})
```

---

## Phase 8: API and Database

### 6.1 API types

**File:** [types.go](bold/api/types.go) (line 20)
- Replace `InboxMaxCount string` with `NextParentChainBlockHash string`

### 6.2 Database schema

**File:** [schema.go](bold/api/db/schema.go) (line 65)
- Replace `InboxMaxCount TEXT` column with `NextParentChainBlockHash TEXT`

### 6.3 Database operations

**File:** [db.go](bold/api/db/db.go) (lines 202, 697, 933)
- Update `WithInboxMaxCount` filter → `WithNextParentChainBlockHash`
- Update insert/update queries

### 6.4 API server

**File:** [methods.go](bold/api/server/methods.go) (lines 48, 70)
- Update query parameter names

---

## Phase 9: Test Updates

Files requiring test updates (all reference `InboxMaxCount` or `BatchLimit` with hardcoded values):

- [poster_catchup_test.go](bold/assertions/poster_catchup_test.go)
- [sync_test.go](bold/assertions/sync_test.go) — lines 158-208, multiple assertions with `InboxMaxCount: big.NewInt(N)`
- [manager_test.go](bold/challenge/manager_test.go) — lines 116, 127
- [tree_test.go](bold/challenge/tree/tree_test.go) — line 33, 321
- [watcher_test.go](bold/challenge/chain/watcher_test.go) — lines 31, 144, 153
- [assertion_chain_test.go](bold/protocol/sol/assertion_chain_test.go) — line 315, 449
- [bold_state_provider_test.go](staker/bold/bold_state_provider_test.go)
- [history_provider_test.go](bold/testing/mocks/state-provider/history_provider_test.go) — line 31
- [prefix_test.go](bold/commitment/proof/prefix/prefix_test.go) — lines 110, 184, 266
- [db_test.go](bold/api/db/db_test.go) — lines 36, 199, 244, 568

---

## Key Risks

| Risk | Mitigation |
|------|-----------|
| `GlobalState.Hash()` mismatch Go vs Solidity vs Rust | Unit tests comparing all three implementations for identical inputs. Key edge cases: all MEL fields zero, only MELStateHash non-zero, both non-zero. The Rust prover uses `bytes32_last_non_zero_index()` for backwards compat — Go and Solidity must match. **The contracts PR (#427) currently hashes all 4 fields unconditionally — this likely needs to be updated to match the Rust approach.** |
| MEL validation not caught up when posting assertions | `ValidatedMELStateLookup` returns `ErrChainCatchingUp`; poster waits and retries |
| Using unvalidated MEL state for assertions | Enforce all MEL state lookups go through `ValidatedMELStateLookup`, never directly via `MessageExtractor` |
| Backwards compatibility for pre-MEL chains | Feature flag or MEL activation check — if MEL not active, fall back to old InboxMaxCount flow |
| Overflow assertion semantics change | PR #427 removes overflow assertions entirely; remove the overflow check in poster.go |
| BOLD challenge machine hashes change due to MEL fields in `GoGlobalState.Hash()` | Pre-MEL states produce the same hash (backwards compatible via non-zero check). Only post-MEL assertions produce new-format hashes, so existing challenges are unaffected |
| MEL preimages not available for BOLD one-step proofs | `CreateReadyValidationEntry` must include MEL preimages from `melValidator.FetchMsgPreimagesAndRelevantState()` |
| Reorgs invalidate validated MEL state | `MELValidator.rewindOnMELReorgs()` already handles this; `ValidatedMELStateLookup` will see reduced `latestValidatedParentChainBlock` after reorg |

---

## Verification

1. **Unit tests**: Run `go test ./bold/...` and `go test ./staker/bold/...` after each phase
2. **Integration tests**: Run `TestValidationPostMEL` and `TestValidationPostMELReorgHandle` from [message_extraction_layer_validation_test.go](system_tests/message_extraction_layer_validation_test.go)
3. **BOLD system test**: Run existing BOLD system tests (search for `TestBold` in system_tests/) to verify assertion posting and challenge flows
4. **Hash consistency**: Write a targeted test that constructs a `GoGlobalState` with MEL fields and verifies the hash matches what the Solidity contract produces
5. **Validation pipeline**: Verify that the BOLD state provider refuses to produce execution states when MEL validation hasn't caught up — `ErrChainCatchingUp` should propagate correctly
6. **Preimage completeness**: Verify that BOLD one-step proofs include all MEL-related preimages by running `CollectProof` against a MEL-enabled test chain and checking the proof validates
7. **End-to-end**: Deploy a local test chain with MEL enabled, post assertions, verify they're accepted by contracts, and run a full challenge lifecycle
