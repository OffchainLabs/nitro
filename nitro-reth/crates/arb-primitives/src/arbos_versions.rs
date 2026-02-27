use alloy_primitives::{Address, Bytes, address, bytes};

/// EIP-2935 history storage address.
pub const HISTORY_STORAGE_ADDRESS: Address =
    address!("0000F90827F1C53a10cb7A02335B175320002935");

/// EIP-2935 history storage contract code for Arbitrum.
pub const HISTORY_STORAGE_CODE_ARBITRUM: Bytes = bytes!("3373fffffffffffffffffffffffffffffffffffffffe1460605760203603605c575f3563a3b1b31d5f5260205f6004601c60645afa15605c575f51600181038211605c57816205ffd0910311605c576205ffd09006545f5260205ff35b5f5ffd5b5f356205ffd0600163a3b1b31d5f5260205f6004601c60645afa15605c575f5103065500");

/// Precompile addresses and the ArbOS version that introduced them.
///
/// During version upgrades, newly introduced precompiles get their code
/// set to `[INVALID]` to mark them as existing accounts.
pub static PRECOMPILE_MIN_ARBOS_VERSIONS: &[(Address, u64)] = &[
    // ArbWasm: introduced in ArbOS 30 (Stylus)
    (address!("0000000000000000000000000000000000000071"), 30),
    // ArbWasmCache: introduced in ArbOS 30
    (address!("0000000000000000000000000000000000000072"), 30),
];

/// ArbOS version identifiers.
///
/// Controls version-gated behavior across the node.
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
#[repr(u64)]
pub enum ArbOSVersion {
    V1 = 1,
    V2 = 2,
    V3 = 3,
    V4 = 4,
    V5 = 5,
    V10 = 10,
    V11 = 11,
    V20 = 20,
    V30 = 30,
    V31 = 31,
    V32 = 32,
    V40 = 40,
    V41 = 41,
    V50 = 50,
    V51 = 51,
    V60 = 60,
}

impl ArbOSVersion {
    pub fn from_u64(v: u64) -> Option<Self> {
        match v {
            1 => Some(Self::V1),
            2 => Some(Self::V2),
            3 => Some(Self::V3),
            4 => Some(Self::V4),
            5 => Some(Self::V5),
            10 => Some(Self::V10),
            11 => Some(Self::V11),
            20 => Some(Self::V20),
            30 => Some(Self::V30),
            31 => Some(Self::V31),
            32 => Some(Self::V32),
            40 => Some(Self::V40),
            41 => Some(Self::V41),
            50 => Some(Self::V50),
            51 => Some(Self::V51),
            60 => Some(Self::V60),
            _ => None,
        }
    }

    pub fn as_u64(self) -> u64 {
        self as u64
    }

    /// Returns true if this version is reserved for Orbit-chain custom upgrades.
    pub fn is_orbit_reserved(version: u64) -> bool {
        matches!(
            version,
            12..=19 | 21..=29 | 33..=39 | 42..=49 | 52..=59
        )
    }
}
