use reth_chainspec::ChainSpec;
pub use reth_chainspec::EthChainSpec;
use revm::primitives::hardfork::SpecId;

/// ArbOS version constants.
///
/// These map to EVM spec upgrades gated by the ArbOS version
/// stored in the block header's mix_hash.
pub mod arbos_version {
    pub const ARBOS_VERSION_2: u64 = 2;
    pub const ARBOS_VERSION_3: u64 = 3;
    pub const ARBOS_VERSION_4: u64 = 4;
    pub const ARBOS_VERSION_5: u64 = 5;
    pub const ARBOS_VERSION_6: u64 = 6;
    pub const ARBOS_VERSION_7: u64 = 7;
    pub const ARBOS_VERSION_8: u64 = 8;
    pub const ARBOS_VERSION_9: u64 = 9;
    pub const ARBOS_VERSION_10: u64 = 10;
    /// ArbOS version 11 — Shanghai EVM rules (PUSH0, etc.).
    pub const ARBOS_VERSION_11: u64 = 11;
    /// Gas for scheduled retry txs is subtracted from parent tx gas used.
    pub const ARBOS_VERSION_FIX_REDEEM_GAS: u64 = ARBOS_VERSION_11;
    /// ArbOS version 20 — Cancun EVM rules (transient storage, blob base fee).
    pub const ARBOS_VERSION_20: u64 = 20;
    /// ArbOS version 30 — Stylus support.
    pub const ARBOS_VERSION_30: u64 = 30;
    pub const ARBOS_VERSION_STYLUS: u64 = ARBOS_VERSION_30;
    /// ArbOS version 31 — Stylus fixes (return data cost check, etc.).
    pub const ARBOS_VERSION_31: u64 = 31;
    pub const ARBOS_VERSION_STYLUS_FIXES: u64 = ARBOS_VERSION_31;
    /// ArbOS version 32 — Stylus charging fixes.
    pub const ARBOS_VERSION_32: u64 = 32;
    pub const ARBOS_VERSION_STYLUS_CHARGING_FIXES: u64 = ARBOS_VERSION_32;
    /// ArbOS version 40 — Prague EVM rules.
    pub const ARBOS_VERSION_40: u64 = 40;
    pub const ARBOS_VERSION_41: u64 = 41;
    /// ArbOS version 50 — Dia upgrade.
    pub const ARBOS_VERSION_50: u64 = 50;
    pub const ARBOS_VERSION_DIA: u64 = ARBOS_VERSION_50;
    /// Maximum ArbOS version supported by this node.
    pub const MAX_ARBOS_VERSION_SUPPORTED: u64 = ARBOS_VERSION_50;
    /// ArbOS version 51 — multi-constraint fix.
    pub const ARBOS_VERSION_MULTI_CONSTRAINT_FIX: u64 = 51;
    /// ArbOS version 60 — multi-gas constraints.
    pub const ARBOS_VERSION_MULTI_GAS_CONSTRAINTS: u64 = 60;
    pub const ARBOS_VERSION_60: u64 = 60;
}

/// Trait for Arbitrum chain specifications.
///
/// Provides the chain ID and version-gated spec ID mapping
/// needed by the EVM configuration layer.
pub trait ArbitrumChainSpec {
    /// Returns the chain ID.
    fn chain_id(&self) -> u64;

    /// Maps a timestamp to the appropriate SpecId.
    fn spec_id_by_timestamp(&self, timestamp: u64) -> SpecId;

    /// Maps an ArbOS version to the appropriate SpecId.
    fn spec_id_by_arbos_version(&self, arbos_version: u64) -> SpecId;
}

/// Map ArbOS version to the appropriate SpecId.
pub fn spec_id_by_arbos_version(arbos_version: u64) -> SpecId {
    if arbos_version >= arbos_version::ARBOS_VERSION_40 {
        SpecId::PRAGUE
    } else if arbos_version >= arbos_version::ARBOS_VERSION_20 {
        SpecId::CANCUN
    } else if arbos_version >= arbos_version::ARBOS_VERSION_11 {
        SpecId::SHANGHAI
    } else {
        SpecId::MERGE
    }
}

/// Arbitrum Sepolia hardfork timestamps.
pub const ARBITRUM_SEPOLIA_SHANGHAI_TIMESTAMP: u64 = 1_706_634_000;
pub const ARBITRUM_SEPOLIA_CANCUN_TIMESTAMP: u64 = 1_709_229_600;
pub const ARBITRUM_SEPOLIA_PRAGUE_TIMESTAMP: u64 = 1_746_543_285;

/// Map timestamp to SpecId for Arbitrum Sepolia.
pub fn arbitrum_sepolia_spec_id_by_timestamp(timestamp: u64) -> SpecId {
    if timestamp >= ARBITRUM_SEPOLIA_PRAGUE_TIMESTAMP {
        SpecId::PRAGUE
    } else if timestamp >= ARBITRUM_SEPOLIA_CANCUN_TIMESTAMP {
        SpecId::CANCUN
    } else if timestamp >= ARBITRUM_SEPOLIA_SHANGHAI_TIMESTAMP {
        SpecId::SHANGHAI
    } else {
        SpecId::MERGE
    }
}

/// Simple Arbitrum chain spec.
#[derive(Clone, Debug, Default)]
pub struct ArbChainSpec {
    pub chain_id: u64,
}

impl ArbitrumChainSpec for ArbChainSpec {
    fn chain_id(&self) -> u64 {
        self.chain_id
    }

    fn spec_id_by_timestamp(&self, timestamp: u64) -> SpecId {
        arbitrum_sepolia_spec_id_by_timestamp(timestamp)
    }

    fn spec_id_by_arbos_version(&self, arbos_version: u64) -> SpecId {
        spec_id_by_arbos_version(arbos_version)
    }
}

/// Blanket implementation for reth's `ChainSpec`.
impl ArbitrumChainSpec for ChainSpec {
    fn chain_id(&self) -> u64 {
        self.chain().id()
    }

    fn spec_id_by_timestamp(&self, timestamp: u64) -> SpecId {
        arbitrum_sepolia_spec_id_by_timestamp(timestamp)
    }

    fn spec_id_by_arbos_version(&self, arbos_version: u64) -> SpecId {
        spec_id_by_arbos_version(arbos_version)
    }
}

/// Arbitrum One chain ID.
pub const ARBITRUM_ONE_CHAIN_ID: u64 = 42161;

/// Arbitrum Sepolia chain ID.
pub const ARBITRUM_SEPOLIA_CHAIN_ID: u64 = 421614;
