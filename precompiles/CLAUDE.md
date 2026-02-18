# Precompiles

ArbOS precompiles are system contracts at fixed addresses. They expose chain management, gas info, retryables, and Stylus APIs to L2 callers.

## Adding a precompile method

A precompile spans three layers that must stay in sync:

1. **Solidity interface** (`contracts/src/precompiles/ArbFoo.sol`) -- defines the ABI. After editing, run `make build-solidity && make contracts` to regenerate Go bindings in `solgen/go/precompilesgen/`.

2. **Go implementation** (`precompiles/ArbFoo.go`) -- implements the methods. The struct must have an `Address addr` field. Methods are matched to ABI functions by reflection via `MakePrecompile` in `precompile.go`:
   - Method name must match the Solidity function name (PascalCase).
   - First param is `ctx` (a `*Context`), second is `mech` (a `*vm.EVM`). Remaining params match the Solidity signature.
   - Return `error` as the last return value.
   - Purity (`pure`/`view`/`write`/`payable`) is inferred from the Solidity ABI.
   - Events are declared as function fields on the struct (e.g. `ChainOwnerAdded func(ctx, mech, common.Address) error`) with a corresponding `GasCost` variant.

3. **Registration** (`precompile.go:Precompiles()`) -- wire it up with `insert(MakePrecompile(precompilesgen.ArbFooMetaData, &ArbFoo{Address: types.ArbFooAddress}))`. The address constant lives in `go-ethereum/core/types/arbitrum_signer.go`.

### Version gating

Methods can be gated to an ArbOS version:
```go
p := insert(MakePrecompile(...))
p.methodsByName["NewMethod"].arbosVersion = params.ArbosVersion_50
```

### Access control wrappers

- `OwnerPrecompile` -- restricts all methods to chain owners (used by ArbOwner)
- `DebugPrecompile` -- only available when `DebugMode()` is enabled (used by ArbDebug)
- Applied in `Precompiles()` via `debugOnly(addr, impl)` or the owner wrapping pattern
