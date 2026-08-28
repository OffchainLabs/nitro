//! Hostio pricing, shared with the prover-side Stylus machinery via `arbutil`.

pub use nitro_arbutil::pricing::*;

/// EVM gas constants used by host functions.
pub mod evm_gas {
    /// params.SstoreSentryGasEIP2200
    pub const SSTORE_SENTRY_GAS: u64 = 2300;
    /// params.ColdAccountAccessCostEIP2929
    pub const COLD_ACCOUNT_GAS: u64 = 2600;
    /// params.ColdSloadCostEIP2929
    pub const COLD_SLOAD_GAS: u64 = 2100;
    /// params.WarmStorageReadCostEIP2929
    pub const WARM_SLOAD_GAS: u64 = 100;
    /// params.WarmStorageReadCostEIP2929 (TLOAD cost)
    pub const TLOAD_GAS: u64 = WARM_SLOAD_GAS;
    /// params.WarmStorageReadCostEIP2929 (TSTORE cost)
    pub const TSTORE_GAS: u64 = WARM_SLOAD_GAS;
    /// params.LogGas
    pub const LOG_TOPIC_GAS: u64 = 375;
    /// params.LogDataGas
    pub const LOG_DATA_GAS: u64 = 8;
    /// Minimum gas the cache requires for SSTORE operations.
    pub const STORAGE_CACHE_REQUIRED_ACCESS_GAS: u64 = 10;
}
