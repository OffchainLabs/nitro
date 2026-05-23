use crate::DatabaseError;
use alloy_primitives::U256;

/// Errors raised by the arb-storage layer.
///
/// `Database(...)` is an infrastructure failure that must abort the calling
/// block. The remaining variants signal that the stored bytes do not match
/// the layout the type expects, which indicates either corrupted state or a
/// version skew between layout code and on-chain data; both abort the block.
#[derive(Debug, thiserror::Error)]
pub enum StorageError {
    /// The underlying state database returned an error.
    #[error(transparent)]
    Database(#[from] DatabaseError),

    /// A stored value violated the layout invariants of its typed wrapper
    /// (e.g. an address slot whose upper 12 bytes are non-zero).
    #[error("invalid storage layout at slot {slot}: {reason}")]
    InvalidLayout {
        /// The slot whose layout is invalid.
        slot: U256,
        /// Short description of the broken invariant.
        reason: &'static str,
    },

    /// A storage container detected a broken structural invariant (e.g. a
    /// queue read past `next_put`).
    #[error("storage invariant violated: {0}")]
    Invariant(&'static str),
}
