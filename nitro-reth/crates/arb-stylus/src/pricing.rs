//! Hostio pricing, shared with the prover-side Stylus machinery via `arbutil`.

pub use nitro_arbutil::pricing::*;

/// EVM gas constants used by host functions.
pub mod evm_gas {
    pub use nitro_arbutil::evm::{
        COLD_ACCOUNT_GAS, COLD_SLOAD_GAS, LOG_DATA_GAS, LOG_TOPIC_GAS, SSTORE_SENTRY_GAS,
        TLOAD_GAS, TSTORE_GAS, WARM_SLOAD_GAS,
    };
}
