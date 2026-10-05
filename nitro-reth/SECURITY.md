# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Arbitrum Reth, please report it
responsibly. Do **not** file a public issue.

**Email:** [til@bloctopus.io](mailto:til@bloctopus.io)

Please include a description of the vulnerability, steps to reproduce, and
any relevant logs or proof of concept.

Response cadence:
- Acknowledgement within 48 hours of receipt.
- Triage and severity assessment within 5 business days.
- Disclosure timeline coordinated with the reporter; default 90 days.

## Supported Versions

- `til/dev`: active development branch; receives security fixes first.
- `master`: stable release branch; receives backported security fixes.

Older release branches are not maintained.

## Security Testing Baseline

The following automated checks run in CI on every pull request and on
pushes to the supported branches:

- `cargo audit`: scans `Cargo.lock` against the RustSec advisory database.
  The ignore list lives in `.cargo/audit.toml` and is mirrored in
  `deny.toml`; every ignored advisory is transitive via reth, wasmer, or
  revm's `ark-*` dependencies and cannot be patched at this layer.
- `cargo deny check`: enforces licence and supply-chain policy
  (allowed licences, banned sources, advisory database).
- `cargo +nightly miri test -p arb-storage --lib` and
  `cargo +nightly miri test -p arb-context`: validate aliasing safety
  on the lifetime-bound `Storage<'a, D>` foundation and the per-block
  `ArbPrecompileCtx`.
- `cargo doc --workspace --no-deps` with `RUSTDOCFLAGS="-D warnings"`:
  rejects broken intra-doc links and missing public-API docs.
- Regression gates (see `.github/workflows/lint.yml`) reject reintroduction
  of the following patterns in the precompile and EVM glue layers:
  `thread_local!` and static `Mutex`/`RwLock`/`OnceCell`
  in `crates/arb-precompiles/src/`, `Result<_, ()>` and `Result<_, String>`
  in the core crates, raw `unsafe { &mut * ... }` derefs in `crates/arb-evm/src/`,
  raw `as *mut State<...>` casts outside `crates/arb-stylus/`, the
  `_via_backend` suffix, and the `with_active`/`install_active`/
  `clear_active`/`ACTIVE_CTX` ambient-context helpers. A contributor who
  reintroduces any of these will see the corresponding `lint.yml` job fail
  with file:line.

All `unsafe` blocks in the workspace carry focused `SAFETY:` comments
explaining the runtime invariant the block relies on.

## Known Structural Compromises

### Wasmer FFI cordon

`crates/arb-stylus/src/evm_api_impl.rs` defines
`unsafe impl Send for StylusEvmApi` and stores a `*mut dyn JournalAccess`
inside the environment passed to wasmer. Both are forced by
`wasmer::FunctionEnv<T>`'s requirement that `T: Send + 'static`; the host
state is logically borrowed from the EVM frame for the duration of the
WASM call but the Wasmer API has no way to express that lifetime. Removing
this compromise requires forking wasmer.

### revm opcode table

`crates/arb-evm/src/evm.rs::POSTER_BALANCE_CORRECTION` is a `thread_local`
that wires per-tx scratch state (poster fee, redeemer, calldata units)
into the `arb_balance` / `arb_selfbalance` opcode handlers. revm's
`Instruction<H>` slot in `[Instruction; 256]` is a bare `fn` pointer that
cannot capture a closure, so the only way to thread per-tx state into the
opcode handler is through a thread-local. This is the single remaining
ambient-state path in `arb-evm`.

### `Storage::state_mut()`

`Storage<'a, D>` stores its backing state as a raw `NonNull<State<D>>`
plus a `PhantomData<&'a mut State<D>>`. The `state_mut()` accessor
materialises a `&'a mut State<D>` from the pointer, which is necessary
because ArbOS storage and EVM-level state mutations must reach the same
underlying `State<D>` while the borrow checker only allows one live
mutable borrow at a time. The compromise is documented at the struct
declaration and on the `state_mut` method; the `arb-storage` `--lib`
miri suite covers the safe paths, while integration tests that hit the
aliasing pattern through the full ArbOS initialisation are excluded from
the miri gate.
