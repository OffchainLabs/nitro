//! Gas and ink units, shared with the prover-side Stylus machinery.
//!
//! `Gas` counts EVM gas; `Ink` counts Stylus computation
//! (1 EVM gas = `ink_price` ink, default 10,000).

pub use nitro_arbutil::evm::api::{Gas, Ink};
