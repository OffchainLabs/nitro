//! Compile-time ABI for Arbitrum precompiles, sourced from the pinned
//! `nitro-precompile-interfaces` submodule so `const SELECTOR` and
//! `SIGNATURE_HASH` are derived from the same `.sol` files Nitro consumes.

#[allow(missing_docs, non_snake_case, non_camel_case_types)]
mod arbsys {
    alloy_sol_types::sol!("nitro-precompile-interfaces/ArbSys.sol");
}

pub use arbsys::ArbSys as IArbSys;

