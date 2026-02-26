use alloy_primitives::{U256, keccak256};

/// Computes a storage slot using the same keccak256-based mapping as Go's mapAddress.
///
/// The algorithm: hash(storage_key || key_bytes[0..31]) || key_bytes[31]
/// This preserves the last byte and hashes only the first 31 bytes.
pub fn storage_key_map(storage_key: &[u8], offset: u64) -> U256 {
    const BOUNDARY: usize = 31;

    let mut key_bytes = [0u8; 32];
    key_bytes[24..32].copy_from_slice(&offset.to_be_bytes());

    let mut data = Vec::with_capacity(storage_key.len() + BOUNDARY);
    data.extend_from_slice(storage_key);
    data.extend_from_slice(&key_bytes[..BOUNDARY]);
    let h = keccak256(&data);

    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key_bytes[BOUNDARY];
    U256::from_be_bytes(mapped)
}

/// Computes a storage slot for an arbitrary B256 key using Go's mapAddress algorithm.
pub fn storage_key_map_b256(storage_key: &[u8], key: &[u8; 32]) -> U256 {
    const BOUNDARY: usize = 31;

    let mut data = Vec::with_capacity(storage_key.len() + BOUNDARY);
    data.extend_from_slice(storage_key);
    data.extend_from_slice(&key[..BOUNDARY]);
    let h = keccak256(&data);

    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key[BOUNDARY];
    U256::from_be_bytes(mapped)
}
