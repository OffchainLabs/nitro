//! Block-level access lists for Reth.
//! [EIP-7928]: <https://eips.ethereum.org/EIPS/eip-7928>
#![cfg_attr(not(feature = "std"), no_std)]

#[allow(unused_imports)]
#[macro_use]
extern crate alloc;

/// Module containing constants used throughout the block access list.
pub mod constants;
pub use constants::*;

/// Module for handling storage changes within a block.
pub mod storage_change;
pub use storage_change::*;

/// Module for managing storage slots and their changes.
pub mod slot_changes;
pub use slot_changes::*;

/// Module for handling balance changes within a block.
pub mod balance_change;
pub use balance_change::*;

/// Module for handling nonce changes within a block.
pub mod nonce_change;
pub use nonce_change::*;

/// Module for handling code changes within a block.
pub mod code_change;
pub use code_change::*;

/// Module for managing account changes within a block.
pub mod account_changes;
pub use account_changes::*;

/// Module for managing block access lists.
pub mod block_access_list;
pub use block_access_list::*;

/// Serde for quantity types.
#[cfg(feature = "serde")]
mod quantity {
    use alloy_primitives::U64;
    use serde::{Deserialize, Deserializer, Serialize, Serializer};

    /// Serializes a primitive number as a "quantity" hex string.
    pub(crate) fn serialize<S>(value: &u64, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        U64::from(*value).serialize(serializer)
    }

    /// Deserializes a primitive number from a "quantity" hex string.
    pub(crate) fn deserialize<'de, D>(deserializer: D) -> Result<u64, D::Error>
    where
        D: Deserializer<'de>,
    {
        U64::deserialize(deserializer).map(|value| value.to())
    }
}
