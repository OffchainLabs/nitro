use alloy_primitives::{keccak256, B256, U256};

pub use arb_storage::{ARBOS_STATE_ADDRESS, FILTERED_TX_STATE_ADDRESS};

/// Subspace keys for ArbOS partitioned storage (matching arbos_state constants).
pub const L1_PRICING_SUBSPACE: &[u8] = &[0];
pub const L2_PRICING_SUBSPACE: &[u8] = &[1];
pub const RETRYABLES_SUBSPACE: &[u8] = &[2];
pub const ADDRESS_TABLE_SUBSPACE: &[u8] = &[3];
pub const CHAIN_OWNER_SUBSPACE: &[u8] = &[4];
pub const PROGRAMS_SUBSPACE: &[u8] = &[8];
pub const NATIVE_TOKEN_SUBSPACE: &[u8] = &[10];
pub const TRANSACTION_FILTERER_SUBSPACE: &[u8] = &[11];

/// Subspace keys within the PROGRAMS subspace.
pub const PROGRAMS_PARAMS_KEY: &[u8] = &[0];
pub const PROGRAMS_DATA_KEY: &[u8] = &[1];
pub const CACHE_MANAGERS_KEY: &[u8] = &[4];

/// Root-level ArbOS state field offsets used outside of the arbos crate.
pub const VERSION_OFFSET: u64 = 0;
pub const CHAIN_ID_OFFSET: u64 = 4;
pub const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;

/// Compute the EVM storage slot for an ArbOS field at a given offset
/// within a storage scope defined by `storage_key`.
///
/// Computes `keccak256(storage_key || key[0..31]) || key[31]`.
pub fn map_slot(storage_key: &[u8], offset: u64) -> U256 {
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

/// Compute the EVM storage slot for a B256 key within a storage scope.
pub fn map_slot_b256(storage_key: &[u8], key: &B256) -> U256 {
    const BOUNDARY: usize = 31;

    let mut buf = [0u8; 64];
    let sk_len = storage_key.len();
    buf[..sk_len].copy_from_slice(storage_key);
    buf[sk_len..sk_len + BOUNDARY].copy_from_slice(&key.0[..BOUNDARY]);
    let h = keccak256(&buf[..sk_len + BOUNDARY]);

    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key.0[BOUNDARY];
    U256::from_be_bytes(mapped)
}

/// Derive a subspace storage key from a parent key and child key bytes.
///
/// Computes `keccak256(parent_key || sub_key)`.
pub fn derive_subspace_key(parent_key: &[u8], sub_key: &[u8]) -> B256 {
    let p_len = parent_key.len();
    let s_len = sub_key.len();
    if p_len + s_len <= 64 {
        let mut buf = [0u8; 64];
        buf[..p_len].copy_from_slice(parent_key);
        buf[p_len..p_len + s_len].copy_from_slice(sub_key);
        keccak256(&buf[..p_len + s_len])
    } else {
        let mut v = Vec::with_capacity(p_len + s_len);
        v.extend_from_slice(parent_key);
        v.extend_from_slice(sub_key);
        keccak256(&v)
    }
}

/// The root storage key for ArbOS state (empty, since base_key is B256::ZERO).
pub const ROOT_STORAGE_KEY: &[u8] = &[];

/// Compute a root-level ArbOS state slot.
#[inline]
pub fn root_slot(offset: u64) -> U256 {
    map_slot(ROOT_STORAGE_KEY, offset)
}

/// Compute a slot within a subspace of the root ArbOS state.
///
/// E.g., `subspace_slot(L1_PRICING_SUBSPACE, field_offset)` for an L1 pricing field.
pub fn subspace_slot(subspace_key: &[u8], offset: u64) -> U256 {
    let sub_storage_key = derive_subspace_key(ROOT_STORAGE_KEY, subspace_key);
    map_slot(sub_storage_key.as_slice(), offset)
}

// ── L2 pricing vector helpers ────────────────────────────────────────

/// L2 pricing subspace key (root → L2_PRICING_SUBSPACE).
fn l2_pricing_subspace() -> B256 {
    derive_subspace_key(ROOT_STORAGE_KEY, L2_PRICING_SUBSPACE)
}

/// Subspace keys within L2 pricing.
const GAS_CONSTRAINTS_SUBKEY: &[u8] = &[0];

/// Derive a sub-storage vector key under L2 pricing.
fn l2_vector_key(sub_key: &[u8]) -> B256 {
    derive_subspace_key(l2_pricing_subspace().as_slice(), sub_key)
}

/// Subspace key for element `index` within a vector.
fn vector_element_key(vector_key: &B256, index: u64) -> B256 {
    derive_subspace_key(vector_key.as_slice(), &index.to_be_bytes())
}

/// Slot for field `offset` within element `index` of a vector.
pub fn vector_element_field(vector_key: &B256, index: u64, offset: u64) -> U256 {
    let elem = vector_element_key(vector_key, index);
    map_slot(elem.as_slice(), offset)
}

/// Gas constraints vector key.
pub fn gas_constraints_vec_key() -> B256 {
    l2_vector_key(GAS_CONSTRAINTS_SUBKEY)
}
