# Stylus hostio reference programs

Each `hostio_*.wat` here is a minimal Stylus contract that calls **one** hostio
and returns the raw result. Programs are deliberately argumentless beyond
their calldata: per-variant numeric inputs, slot keys, payload bytes, etc.
are supplied by the JSON fixture, not baked into the WAT. This keeps each
hostio's coverage to one source-of-truth program, and pushes per-call
parameterisation up into the fixture authors (Stage 3 agents D1-D4).

## Conventions

- **Imports** come from the `vm_hooks` namespace and must match arbreth's
  registration in `crates/arb-stylus/src/native.rs`. Names line up with the
  Nitro `arbitrator/wasm-libraries/user-host` exports.
- Every program imports `vm_hooks::read_args` and `vm_hooks::write_result`
  alongside the hostio under test, exports a single `user_entrypoint
  (param $args_len i32) (result i32)`, and exports a single-page `memory`.
  Status code 0 = Ok, non-zero = Revert.
- Calldata is laid out at memory offset 0 by `read_args`. Return payloads are
  written from a higher offset (typically `0x100`) when the hostio result
  shares space with the input; a zero-length return is fine.
- One hostio per file. Programs that need multiple hostios to express a single
  semantic op (`storage_cache_bytes32` + `storage_flush_cache` for "store") are
  documented as such in the file header.

## Fixture-author workflow

Each `hostio_<name>.wat` is the program input for ~3-5 JSON fixtures under
`fixtures/stylus/hostio/<name>/<variant>.json`. The fixture supplies:

1. `wasm`: the compiled bytes of the WAT (or a reference to compile it).
2. `calldata`: hex-encoded input matching the layout documented at the top
   of the WAT.
3. `expected.return_data`, `expected.gas_used`, `expected.ink_used`,
   `expected.logs`, `expected.storage_writes`, `expected.multi_gas` (v60+).

The runner deploys the program, invokes it with the calldata, and asserts
the captured fields against a Nitro reference run. Variant counts target the
gas-pricing branches (cold/warm, ArbOS-version-gated paths, size-tier
thresholds), not exhaustive value coverage.

## Adding a new hostio

1. Copy `template_hostio.wat` and rename to `hostio_<name>.wat`.
2. Replace the `<HOSTIO>` placeholder with the import name and signature
   (cross-check against `arb-stylus::native::imports!()`).
3. Document the calldata layout, return shape, and 3-5 useful variants in
   the header comment.
4. Smoke-test: `wat::parse_str(include_str!(...))` should return `Ok`. The
   fixture-runner CI gate compiles every `_wat/*.wat` before any fixture
   referencing it is exercised.
5. Land the WAT in one commit; fixtures referencing it follow.
