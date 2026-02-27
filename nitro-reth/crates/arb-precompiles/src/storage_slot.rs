use alloy_primitives::{keccak256, B256, U256};

/// ArbOS state backing address.
pub const ARBOS_STATE_ADDRESS: alloy_primitives::Address = alloy_primitives::Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x0a, 0x4b, 0x05,
]);

/// Subspace keys for ArbOS partitioned storage (matching arbos_state constants).
pub const L1_PRICING_SUBSPACE: &[u8] = &[0];
pub const L2_PRICING_SUBSPACE: &[u8] = &[1];
pub const RETRYABLES_SUBSPACE: &[u8] = &[2];
pub const ADDRESS_TABLE_SUBSPACE: &[u8] = &[3];
pub const CHAIN_OWNER_SUBSPACE: &[u8] = &[4];
pub const SEND_MERKLE_SUBSPACE: &[u8] = &[5];
pub const BLOCKHASHES_SUBSPACE: &[u8] = &[6];
pub const PROGRAMS_SUBSPACE: &[u8] = &[7];
pub const CHAIN_CONFIG_SUBSPACE: &[u8] = &[8];
pub const FEATURES_SUBSPACE: &[u8] = &[9];
pub const NATIVE_TOKEN_SUBSPACE: &[u8] = &[10];
pub const TRANSACTION_FILTERER_SUBSPACE: &[u8] = &[11];

/// Subspace keys within the PROGRAMS subspace.
pub const PROGRAMS_PARAMS_KEY: &[u8] = &[0];
pub const CACHE_MANAGERS_KEY: &[u8] = &[4];

/// Cache managers subspace within ArbOS (PROGRAMS → CACHE_MANAGERS).
/// Not a direct root subspace; derive at runtime via `programs_cache_managers_key()`.
pub const CACHE_MANAGERS_SUBSPACE: &[u8] = CACHE_MANAGERS_KEY;

/// Filtered transactions backing storage account (separate from ArbOS state).
pub const FILTERED_TX_STATE_ADDRESS: alloy_primitives::Address = alloy_primitives::Address::new([
    0xa4, 0xb0, 0x50, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x01,
]);

/// Root-level ArbOS state field offsets (matches Go iota in arbosstate.go).
pub const VERSION_OFFSET: u64 = 0;
pub const UPGRADE_VERSION_OFFSET: u64 = 1;
pub const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;
pub const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
pub const CHAIN_ID_OFFSET: u64 = 4;
pub const GENESIS_BLOCK_NUM_OFFSET: u64 = 5;
pub const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
pub const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
pub const NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET: u64 = 8;
pub const TX_FILTERING_ENABLED_FROM_TIME_OFFSET: u64 = 9;
pub const FILTERED_FUNDS_RECIPIENT_OFFSET: u64 = 10;

/// Compute the EVM storage slot for an ArbOS field at a given offset
/// within a storage scope defined by `storage_key`.
///
/// Matches Go's `mapAddress`: `keccak256(storage_key || key[0..31]) || key[31]`.
pub fn map_slot(storage_key: &[u8], offset: u64) -> U256 {
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

/// Compute the EVM storage slot for a B256 key within a storage scope.
pub fn map_slot_b256(storage_key: &[u8], key: &B256) -> U256 {
    const BOUNDARY: usize = 31;

    let mut data = Vec::with_capacity(storage_key.len() + BOUNDARY);
    data.extend_from_slice(storage_key);
    data.extend_from_slice(&key.0[..BOUNDARY]);
    let h = keccak256(&data);

    let mut mapped = [0u8; 32];
    mapped[..BOUNDARY].copy_from_slice(&h.0[..BOUNDARY]);
    mapped[BOUNDARY] = key.0[BOUNDARY];
    U256::from_be_bytes(mapped)
}

/// Derive a subspace storage key from a parent key and child key bytes.
///
/// Matches Go's `OpenSubStorage`: `keccak256(parent_key || sub_key)`.
pub fn derive_subspace_key(parent_key: &[u8], sub_key: &[u8]) -> B256 {
    let mut combined = Vec::with_capacity(parent_key.len() + sub_key.len());
    combined.extend_from_slice(parent_key);
    combined.extend_from_slice(sub_key);
    keccak256(&combined)
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

// ── Per-tx scratch slot ──────────────────────────────────────────────

/// Scratch slot used to pass per-transaction L1 poster fee from the
/// executor into the EVM where the ArbGasInfo precompile can read it.
/// The value is written before EVM execution and has no long-term
/// significance — it's overwritten every transaction.
pub const CURRENT_TX_POSTER_FEE_OFFSET: u64 = 255;

/// Compute the storage slot for the per-tx poster fee.
pub fn current_tx_poster_fee_slot() -> U256 {
    map_slot(ROOT_STORAGE_KEY, CURRENT_TX_POSTER_FEE_OFFSET)
}

// ── L2 pricing vector helpers ────────────────────────────────────────

/// L2 pricing subspace key (root → L2_PRICING_SUBSPACE).
pub fn l2_pricing_subspace() -> B256 {
    derive_subspace_key(ROOT_STORAGE_KEY, L2_PRICING_SUBSPACE)
}

/// Subspace keys within L2 pricing (matching Go l2pricing constants).
const GAS_CONSTRAINTS_SUBKEY: &[u8] = &[0];
const MULTI_GAS_CONSTRAINTS_SUBKEY: &[u8] = &[1];
const MULTI_GAS_BASE_FEES_SUBKEY: &[u8] = &[2];

/// Derive a sub-storage vector key under L2 pricing.
fn l2_vector_key(sub_key: &[u8]) -> B256 {
    derive_subspace_key(l2_pricing_subspace().as_slice(), sub_key)
}

/// Slot for the length of a sub-storage vector.
pub fn vector_length_slot(vector_key: &B256) -> U256 {
    map_slot(vector_key.as_slice(), 0)
}

/// Subspace key for element `index` within a vector.
pub fn vector_element_key(vector_key: &B256, index: u64) -> B256 {
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

/// Multi-gas constraints vector key.
pub fn multi_gas_constraints_vec_key() -> B256 {
    l2_vector_key(MULTI_GAS_CONSTRAINTS_SUBKEY)
}

/// Multi-gas base fees subspace key.
pub fn multi_gas_base_fees_subspace() -> B256 {
    l2_vector_key(MULTI_GAS_BASE_FEES_SUBKEY)
}

