# ArbOS

ArbOS is the L2 operating system. All state lives in EVM storage slots of a fictional account (`types.ArbosStateAddress`), accessed through the `storage.Storage` abstraction.

## Storage layer (`storage/`)

State is organized as a tree of sub-spaces. Each sub-space has a `storageKey` (byte slice). Child keys are derived by hashing: `childKey = keccak256(parentKey, childName)`. The root has an empty key.

Slots use a "page" system: the top 31 bytes of a key are hashed (the page number), the bottom byte is preserved (offset within page). This gives 256 contiguous slots per page.

### Typed wrappers

Subsystems access fields through typed wrappers created from fixed offsets:

```go
myField := sto.OpenStorageBackedUint64(offsetN)
val, _ := myField.Get()
_ = myField.Set(newVal)
```

Available types: `StorageBackedUint16/24/32/64`, `StorageBackedBigUint/BigInt`, `StorageBackedInt64`, `StorageBackedBips/UBips`, `StorageBackedAddress`, `StorageBackedAddressOrNil`, `StorageBackedBytes`.

### Collections

- `Queue` -- FIFO backed by sequential slot indices. Used by retryable timeout queue.
- `SubStorageVector` -- dynamic array of sub-storage spaces. Used by L2 pricing constraints.

### Caching

- `OpenCachedSubStorage` -- uses a global LRU hash cache (for hot paths like pricing, retryables)
- `OpenSubStorage` -- no cache (for infrequently accessed data)
- `.WithoutCache()` -- strips cache from collections that access many slots

### Gas accounting

Every `Get` and `Set` burns gas (defined as constants in `storage.go`). Charged via the `burn.Burner` interface threaded through all operations.

## State structure (`arbosState/`)

`ArbosState` is the root. Opened via `OpenArbosState(stateDB, burner)` on every transaction. Top-level fields use fixed `iota` offsets. Each subsystem (l1pricing, l2pricing, retryables, programs, etc.) gets its own sub-space -- see the `SubspaceID` constants in `arbosstate.go`.

### Init vs Open

- `InitializeArbosState` -- called once at genesis. Writes version, initializes all subsystems, runs version upgrades.
- `OpenArbosState` -- called every transaction. Reads version from storage.

### Version upgrades

`UpgradeArbosVersion` steps through versions one at a time, running per-version migration logic. Versions 12-19, 21-29, 33-39, 42-49, 52-59 are reserved for Orbit chain custom upgrades. Upgrades are scheduled via `ScheduleArbOSUpgrade(version, timestamp)` and applied at block start.

## Block processing (`block_processor.go`)

Every block begins with an `InternalTxStartBlock` transaction that:
1. Records the L1 block number and previous block hash
2. Reaps up to 2 expired retryable tickets
3. Updates the L2 pricing model
4. Checks for scheduled ArbOS upgrades

User transactions are then processed. Each tx goes through:
1. Pre-filter hooks (sequencer-specific)
2. L1 poster cost computation (compressed calldata cost in L2 gas units)
3. Gas splitting: `tx.Gas()` -> `dataGas` (L1 cost) + `computeGas`
4. EVM execution via geth's `ApplyTransaction`
5. Post-filter hooks

### Transaction types

- `ArbitrumDepositTx` -- mints ETH (L1->L2 deposit)
- `ArbitrumSubmitRetryableTx` -- creates a retryable ticket
- `ArbitrumRetryTx` -- redeems a retryable
- `ArbitrumInternalTx` -- system operations (start block, batch posting report)

## Key patterns

**Initialize/Open duality**: every subsystem has `InitializeFoo(sto)` (genesis) and `OpenFoo(sto)` (every access). Always called with the subsystem's dedicated sub-storage.

**Offset-based layout**: fields are declared as `iota` offsets, wrapped via `sto.OpenStorageBackedXxx(offset)` in the `Open` function.

**Version gating**: subsystems cache `ArbosVersion` as a plain uint64 and gate behavior behind version checks.

**Burner threading**: all storage operations take a `burn.Burner` for gas accounting. `SystemBurner` (unlimited gas) is used for internal operations. User-facing precompile calls use a burner that charges against the EVM gas budget.
