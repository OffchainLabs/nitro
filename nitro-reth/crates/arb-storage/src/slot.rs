use alloy_primitives::{keccak256, B256, U256};

/// Computes a storage slot using the keccak256-based mapAddress algorithm.
///
/// The algorithm: hash(storage_key || key_bytes[0..31]) || key_bytes[31]
/// This preserves the last byte and hashes only the first 31 bytes.
pub fn storage_key_map(storage_key: &[u8], offset: u64) -> U256 {
    const BOUNDARY: usize = 31;

    let mut key_bytes = [0u8; 32];
    key_bytes[24..32].copy_from_slice(&offset.to_be_bytes());

    let mut buf = [0u8; 64];
    let sk_len = storage_key.len();
    buf[..sk_len].copy_from_slice(storage_key);
    buf[sk_len..sk_len + BOUNDARY].copy_from_slice(&key_bytes[..BOUNDARY]);
    let h = keccak256(&buf[..sk_len + BOUNDARY]);

    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key_bytes[BOUNDARY];
    U256::from_be_bytes(mapped)
}

/// Computes a storage slot for an arbitrary B256 key using the mapAddress algorithm.
pub fn storage_key_map_b256(storage_key: &[u8], key: &[u8; 32]) -> U256 {
    const BOUNDARY: usize = 31;

    let mut buf = [0u8; 64];
    let sk_len = storage_key.len();
    buf[..sk_len].copy_from_slice(storage_key);
    buf[sk_len..sk_len + BOUNDARY].copy_from_slice(&key[..BOUNDARY]);
    let h = keccak256(&buf[..sk_len + BOUNDARY]);

    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key[BOUNDARY];
    U256::from_be_bytes(mapped)
}

/// Derive a sub-storage key: `keccak256(parent_key || sub_key)`.
///
/// Uses an inline stack buffer to avoid heap allocation. For hot paths where
/// `sub_key` is a compile-time constant, prefer caching the result.
pub fn derive_sub_key(parent_key: B256, sub_key: &[u8]) -> B256 {
    let base_slice: &[u8] = if parent_key == B256::ZERO {
        &[]
    } else {
        parent_key.as_slice()
    };
    let base_len = base_slice.len();
    let sub_len = sub_key.len();
    if base_len + sub_len <= 64 {
        let mut buf = [0u8; 64];
        buf[..base_len].copy_from_slice(base_slice);
        buf[base_len..base_len + sub_len].copy_from_slice(sub_key);
        keccak256(&buf[..base_len + sub_len])
    } else {
        let mut v = Vec::with_capacity(base_len + sub_len);
        v.extend_from_slice(base_slice);
        v.extend_from_slice(sub_key);
        keccak256(&v)
    }
}
