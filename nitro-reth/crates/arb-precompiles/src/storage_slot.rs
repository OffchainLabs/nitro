use alloy_primitives::{keccak256, U256};

/// ArbOS state backing address.
pub const ARBOS_STATE_ADDRESS: alloy_primitives::Address = alloy_primitives::Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x0a, 0x4b, 0x05,
]);

/// ArbOS storage space offsets (matching Go's arbosState offsets).
pub const L1_PRICING_SPACE: u64 = 7;
pub const L2_PRICING_SPACE: u64 = 8;

/// Compute the storage slot for an ArbOS state field.
///
/// This matches Go Nitro's `storage.NewSlot` which hashes
/// `"Arbitrum internal storage" || parent_slot` to derive subspace keys.
pub fn compute_storage_slot(parents: &[U256], offset: u64) -> U256 {
    if parents.is_empty() {
        return U256::from(offset);
    }

    let parent = parents[0];
    let mut data = Vec::with_capacity(56);
    data.extend_from_slice(b"Arbitrum internal storage");
    data.extend_from_slice(&parent.to_be_bytes::<32>());

    let base = U256::from_be_bytes(keccak256(&data).0);
    base.wrapping_add(U256::from(offset))
}
