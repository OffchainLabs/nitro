### Internal
- `arb-stylus` now re-exports `Gas`/`Ink`, the hostio pricing constants, and the EVM gas constants from `arbutil` instead of duplicating them; the gas-metering trait is `Gas`-typed and the unused `EvmApiStatus` enum was removed
