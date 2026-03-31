//! Contains the [`StorageChange`] struct, which represents a single storage write operation within
//! a transaction.

use crate::BlockAccessIndex;
use alloy_primitives::U256;

/// Represents a single storage write operation within a transaction.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
#[cfg_attr(feature = "rlp", derive(alloy_rlp::RlpEncodable, alloy_rlp::RlpDecodable))]
#[cfg_attr(feature = "serde", derive(serde::Serialize, serde::Deserialize))]
#[cfg_attr(feature = "borsh", derive(borsh::BorshSerialize, borsh::BorshDeserialize))]
#[cfg_attr(feature = "serde", serde(rename_all = "camelCase"))]
#[cfg_attr(feature = "arbitrary", derive(arbitrary::Arbitrary))]
pub struct StorageChange {
    /// Index of the bal that stores the performed write.
    #[cfg_attr(feature = "serde", serde(alias = "txIndex", with = "crate::quantity"))]
    pub block_access_index: BlockAccessIndex,
    /// The new value written to the storage slot.
    #[cfg_attr(feature = "serde", serde(alias = "postValue"))]
    pub new_value: U256,
}

impl StorageChange {
    /// Creates a new `StorageChange`.
    #[inline]
    pub const fn new(block_access_index: BlockAccessIndex, new_value: U256) -> Self {
        Self { block_access_index, new_value }
    }

    /// Returns true if the new value is zero.
    #[inline]
    pub fn is_zero(&self) -> bool {
        self.new_value.is_zero()
    }

    /// Returns true if this change was made by the given transaction.
    #[inline]
    pub const fn is_from_tx(&self, block_index: BlockAccessIndex) -> bool {
        self.block_access_index == block_index
    }

    /// Returns a copy with a different storage value.
    #[inline]
    pub const fn with_value(&self, value: U256) -> Self {
        Self { block_access_index: self.block_access_index, new_value: value }
    }
}
