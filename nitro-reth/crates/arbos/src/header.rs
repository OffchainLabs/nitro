use alloy_primitives::{keccak256, Address, B256, U256};

/// ArbOS state storage address.
pub const ARBOS_STATE_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[0] = 0xA4;
    bytes[1] = 0xB0;
    bytes[2] = 0x5F;
    bytes[3] = 0xFF;
    bytes[4] = 0xFF;
    bytes[5] = 0xFF;
    bytes[6] = 0xFF;
    bytes[7] = 0xFF;
    bytes[8] = 0xFF;
    bytes[9] = 0xFF;
    bytes[10] = 0xFF;
    bytes[11] = 0xFF;
    bytes[12] = 0xFF;
    bytes[13] = 0xFF;
    bytes[14] = 0xFF;
    bytes[15] = 0xFF;
    bytes[16] = 0xFF;
    bytes[17] = 0xFF;
    bytes[18] = 0xFF;
    bytes[19] = 0xFF;
    Address::new(bytes)
};

/// Holds Arbitrum-specific header metadata.
#[derive(Debug, Clone, Default)]
pub struct ArbHeaderInfo {
    /// Merkle root of sent messages.
    pub send_root: B256,
    /// Number of messages sent.
    pub send_count: u64,
    /// Corresponding L1 block number.
    pub l1_block_number: u64,
    /// ArbOS format version.
    pub arbos_format_version: u64,
}

impl ArbHeaderInfo {
    /// Compute the mix_hash from send_count, l1_block_number, and arbos_version.
    pub fn compute_mix_hash(&self) -> B256 {
        compute_nitro_mixhash(self.send_count, self.l1_block_number, self.arbos_format_version)
    }
}

/// Compute the Nitro mix hash from the three u64 components.
///
/// Layout: [send_count (8 bytes)][l1_block_number (8 bytes)][arbos_version (8 bytes)][0..0]
pub fn compute_nitro_mixhash(send_count: u64, l1_block_number: u64, arbos_version: u64) -> B256 {
    let mut mix = [0u8; 32];
    mix[0..8].copy_from_slice(&send_count.to_be_bytes());
    mix[8..16].copy_from_slice(&l1_block_number.to_be_bytes());
    mix[16..24].copy_from_slice(&arbos_version.to_be_bytes());
    B256::from(mix)
}

/// Extract the send root from the first 32 bytes of header extra_data.
pub fn extract_send_root_from_header_extra(extra: &[u8]) -> B256 {
    if extra.len() >= 32 {
        B256::from_slice(&extra[..32])
    } else {
        B256::ZERO
    }
}

/// Extract ArbOS version from header mix_hash (bytes 16-23).
pub fn extract_arbos_version_from_mix_hash(mix_hash: B256) -> u64 {
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&mix_hash.0[16..24]);
    u64::from_be_bytes(buf)
}

/// Extract send count from header mix_hash (bytes 0-7).
pub fn extract_send_count_from_mix_hash(mix_hash: B256) -> u64 {
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&mix_hash.0[0..8]);
    u64::from_be_bytes(buf)
}

/// Extract L1 block number from header mix_hash (bytes 8-15).
pub fn extract_l1_block_number_from_mix_hash(mix_hash: B256) -> u64 {
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&mix_hash.0[8..16]);
    u64::from_be_bytes(buf)
}

/// Convert a u64 to a left-padded B256 (big-endian in last 8 bytes).
fn uint_to_hash_u64_be(k: u64) -> B256 {
    let mut out = [0u8; 32];
    out[24..32].copy_from_slice(&k.to_be_bytes());
    B256::from(out)
}

/// Map a storage key + sub-key to a derived storage slot.
fn storage_key_map(storage_key: &[u8], key: B256) -> B256 {
    let boundary = 31usize;
    let mut data = Vec::with_capacity(storage_key.len() + boundary);
    data.extend_from_slice(storage_key);
    data.extend_from_slice(&key.0[..boundary]);
    let h = keccak256(&data);
    let mut mapped = [0u8; 32];
    mapped[..boundary].copy_from_slice(&h.0[..boundary]);
    mapped[boundary] = key.0[boundary];
    B256::from(mapped)
}

/// Derive a subspace key from parent + id.
fn subspace(parent: &[u8], id: &[u8]) -> [u8; 32] {
    let mut data = Vec::with_capacity(parent.len() + id.len());
    data.extend_from_slice(parent);
    data.extend_from_slice(id);
    keccak256(&data).0
}

/// Calculate the number of partials in the Merkle accumulator.
fn calc_num_partials(size: u64) -> u64 {
    if size == 0 {
        return 0;
    }
    64 - size.leading_zeros() as u64
}

/// Read a u64 from storage at a given slot (big-endian in last 8 bytes).
pub fn read_storage_u64_be<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
    addr: Address,
    slot: B256,
) -> Option<u64> {
    let val = read_slot(addr, slot)?;
    let bytes: [u8; 32] = val.to_be_bytes::<32>();
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&bytes[24..32]);
    Some(u64::from_be_bytes(buf))
}

/// Read a B256 hash from storage at a given slot.
pub fn read_storage_hash<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
    addr: Address,
    slot: B256,
) -> Option<B256> {
    let val = read_slot(addr, slot)?;
    let bytes: [u8; 32] = val.to_be_bytes::<32>();
    Some(B256::from(bytes))
}

/// Compute the Merkle root from partials stored in state.
pub fn merkle_root_from_partials<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
    addr: Address,
    send_merkle_storage_key: &[u8],
    size: u64,
) -> Option<B256> {
    if size == 0 {
        return Some(B256::ZERO);
    }
    let mut hash_so_far: Option<B256> = None;
    let mut capacity_in_hash: u64 = 0;
    let mut capacity = 1u64;
    let num_partials = calc_num_partials(size);
    for level in 0..num_partials {
        let key = uint_to_hash_u64_be(2 + level);
        let slot = storage_key_map(send_merkle_storage_key, key);
        let partial = read_storage_hash(read_slot, addr, slot).unwrap_or(B256::ZERO);
        if partial != B256::ZERO {
            if let Some(mut h) = hash_so_far {
                while capacity_in_hash < capacity {
                    let combined = [h.0.as_slice(), &[0u8; 32]].concat();
                    h = keccak256(&combined);
                    capacity_in_hash *= 2;
                }
                let combined = [partial.0.as_slice(), h.0.as_slice()].concat();
                hash_so_far = Some(keccak256(&combined));
                capacity_in_hash = 2 * capacity;
            } else {
                hash_so_far = Some(partial);
                capacity_in_hash = capacity;
            }
        }
        capacity = capacity.saturating_mul(2);
    }
    hash_so_far
}

/// Derive ArbHeaderInfo from storage reads.
pub fn derive_arb_header_info<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
) -> Option<ArbHeaderInfo> {
    let addr = ARBOS_STATE_ADDRESS;
    let root_storage_key: &[u8] = &[];

    let version_slot = storage_key_map(root_storage_key, uint_to_hash_u64_be(0));
    let arbos_version = read_storage_u64_be(read_slot, addr, version_slot)?;

    let send_merkle_sub = subspace(root_storage_key, &[5u8]);
    let blockhashes_sub = subspace(root_storage_key, &[6u8]);

    let send_count_slot = storage_key_map(&send_merkle_sub, uint_to_hash_u64_be(0));
    let send_count = read_storage_u64_be(read_slot, addr, send_count_slot).unwrap_or(0);

    let send_root =
        merkle_root_from_partials(read_slot, addr, &send_merkle_sub, send_count)
            .unwrap_or(B256::ZERO);

    let l1_block_num_slot = storage_key_map(&blockhashes_sub, uint_to_hash_u64_be(0));
    let l1_block_number =
        read_storage_u64_be(read_slot, addr, l1_block_num_slot).unwrap_or(0);

    Some(ArbHeaderInfo {
        send_root,
        send_count,
        l1_block_number,
        arbos_format_version: arbos_version,
    })
}

/// Get the storage address and slot for the ArbOS L1 block number.
pub fn arbos_l1_block_number_slot() -> (Address, B256) {
    let addr = ARBOS_STATE_ADDRESS;
    let root_storage_key: &[u8] = &[];
    let blockhashes_sub = subspace(root_storage_key, &[6u8]);
    let l1_block_num_slot = storage_key_map(&blockhashes_sub, uint_to_hash_u64_be(0));
    (addr, l1_block_num_slot)
}

/// Read ArbOS version from storage.
pub fn read_arbos_version<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
) -> Option<u64> {
    let addr = ARBOS_STATE_ADDRESS;
    let root_storage_key: &[u8] = &[];
    let version_slot = storage_key_map(root_storage_key, uint_to_hash_u64_be(0));
    read_storage_u64_be(read_slot, addr, version_slot)
}

/// Read the L2 per-block gas limit from storage.
pub fn read_l2_per_block_gas_limit<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
) -> Option<u64> {
    let addr = ARBOS_STATE_ADDRESS;
    let root_storage_key: &[u8] = &[];
    let l2_pricing_subspace = subspace(root_storage_key, &[1u8]);
    let per_block_gas_limit_slot = storage_key_map(&l2_pricing_subspace, uint_to_hash_u64_be(1));
    read_storage_u64_be(read_slot, addr, per_block_gas_limit_slot)
}

/// Read the L2 base fee from storage.
pub fn read_l2_base_fee<F: Fn(Address, B256) -> Option<U256>>(
    read_slot: &F,
) -> Option<u64> {
    let addr = ARBOS_STATE_ADDRESS;
    let root_storage_key: &[u8] = &[];
    let l2_pricing_subspace = subspace(root_storage_key, &[1u8]);
    let price_per_unit_slot = storage_key_map(&l2_pricing_subspace, uint_to_hash_u64_be(2));
    read_storage_u64_be(read_slot, addr, price_per_unit_slot)
}
