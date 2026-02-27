use alloy_primitives::{keccak256, B256};
use revm::Database;

use arb_storage::{Storage, StorageBackedUint64};

/// Event emitted when a Merkle tree node is updated during append.
#[derive(Debug, Clone)]
pub struct MerkleTreeNodeEvent {
    pub level: u64,
    pub num_leaves: u64,
    pub hash: B256,
}

/// Storage-backed Merkle accumulator.
pub struct MerkleAccumulator<D> {
    backing_storage: Storage<D>,
    size: StorageBackedUint64<D>,
}

pub fn initialize_merkle_accumulator<D: Database>(_sto: &Storage<D>) {
    // no-op
}

pub fn open_merkle_accumulator<D: Database>(sto: Storage<D>) -> MerkleAccumulator<D> {
    let size = StorageBackedUint64::new(sto.state_ptr(), sto.base_key(), 0);
    MerkleAccumulator {
        backing_storage: sto,
        size,
    }
}

/// Returns the number of partial tree hashes needed for a given size.
/// This is the bit-length of `size` (i.e. floor(log2(size)) + 1).
pub fn calc_num_partials(size: u64) -> u64 {
    if size == 0 {
        return 0;
    }
    64 - size.leading_zeros() as u64
}

impl<D: Database> MerkleAccumulator<D> {
    fn get_partial(&self, level: u64) -> Result<B256, ()> {
        self.backing_storage.get_by_uint64(2 + level)
    }

    fn set_partial(&self, level: u64, val: B256) -> Result<(), ()> {
        self.backing_storage.set_by_uint64(2 + level, val)
    }

    pub fn append(&self, item_hash: B256) -> Result<Vec<MerkleTreeNodeEvent>, ()> {
        let current_size = self.size.get()?;
        let new_size = current_size + 1;
        self.size.set(new_size)?;

        let mut events = Vec::new();
        let mut level = 0u64;
        let mut so_far = keccak256(item_hash.as_slice());

        loop {
            if level == calc_num_partials(current_size) {
                self.set_partial(level, so_far)?;
                return Ok(events);
            }

            let this_level = self.get_partial(level)?;
            if this_level == B256::ZERO {
                self.set_partial(level, so_far)?;
                return Ok(events);
            }

            let mut combined = Vec::with_capacity(64);
            combined.extend_from_slice(this_level.as_slice());
            combined.extend_from_slice(so_far.as_slice());
            so_far = keccak256(&combined);

            self.set_partial(level, B256::ZERO)?;

            level += 1;
            events.push(MerkleTreeNodeEvent {
                level,
                num_leaves: new_size - 1,
                hash: so_far,
            });
        }
    }

    pub fn size(&self) -> Result<u64, ()> {
        self.size.get()
    }

    pub fn root(&self) -> Result<B256, ()> {
        let size = self.size.get()?;
        if size == 0 {
            return Ok(B256::ZERO);
        }

        let mut hash_so_far: Option<B256> = None;
        let mut capacity_in_hash = 0u64;
        let mut capacity = 1u64;

        for level in 0..calc_num_partials(size) {
            let partial = self.get_partial(level)?;
            if partial != B256::ZERO {
                if let Some(ref mut current) = hash_so_far {
                    while capacity_in_hash < capacity {
                        let mut combined = Vec::with_capacity(64);
                        combined.extend_from_slice(current.as_slice());
                        combined.extend_from_slice(&[0u8; 32]);
                        *current = keccak256(&combined);
                        capacity_in_hash *= 2;
                    }

                    let mut combined = Vec::with_capacity(64);
                    combined.extend_from_slice(partial.as_slice());
                    combined.extend_from_slice(current.as_slice());
                    *current = keccak256(&combined);
                    capacity_in_hash = 2 * capacity;
                } else {
                    hash_so_far = Some(partial);
                    capacity_in_hash = capacity;
                }
            }
            capacity *= 2;
        }

        Ok(hash_so_far.unwrap_or(B256::ZERO))
    }

    pub fn get_partials(&self) -> Result<Vec<B256>, ()> {
        let size = self.size.get()?;
        let num = calc_num_partials(size);
        let mut partials = Vec::with_capacity(num as usize);
        for i in 0..num {
            partials.push(self.get_partial(i)?);
        }
        Ok(partials)
    }

    pub fn state_for_export(&self) -> Result<(u64, B256, Vec<B256>), ()> {
        let root = self.root()?;
        let size = self.size.get()?;
        let partials = self.get_partials()?;
        Ok((size, root, partials))
    }
}

/// In-memory (non-persistent) Merkle accumulator for export/import and testing.
pub struct InMemoryMerkleAccumulator {
    size: u64,
    partials: Vec<B256>,
}

impl InMemoryMerkleAccumulator {
    pub fn new() -> Self {
        Self {
            size: 0,
            partials: Vec::new(),
        }
    }

    pub fn from_partials(partials: Vec<B256>) -> Self {
        let mut size = 0u64;
        let mut level_size = 1u64;
        for p in &partials {
            if *p != B256::ZERO {
                size += level_size;
            }
            level_size *= 2;
        }
        Self { size, partials }
    }

    pub fn size(&self) -> u64 {
        self.size
    }

    fn get_partial(&self, level: u64) -> B256 {
        self.partials
            .get(level as usize)
            .copied()
            .unwrap_or(B256::ZERO)
    }

    fn set_partial(&mut self, level: u64, val: B256) {
        let idx = level as usize;
        if idx >= self.partials.len() {
            self.partials.resize(idx + 1, B256::ZERO);
        }
        self.partials[idx] = val;
    }

    pub fn append(&mut self, item_hash: B256) -> Vec<MerkleTreeNodeEvent> {
        let current_size = self.size;
        self.size += 1;
        let new_size = self.size;

        let mut events = Vec::new();
        let mut level = 0u64;
        let mut so_far = keccak256(item_hash.as_slice());

        loop {
            if level == calc_num_partials(current_size) {
                self.set_partial(level, so_far);
                return events;
            }

            let this_level = self.get_partial(level);
            if this_level == B256::ZERO {
                self.set_partial(level, so_far);
                return events;
            }

            let mut combined = Vec::with_capacity(64);
            combined.extend_from_slice(this_level.as_slice());
            combined.extend_from_slice(so_far.as_slice());
            so_far = keccak256(&combined);

            self.set_partial(level, B256::ZERO);

            level += 1;
            events.push(MerkleTreeNodeEvent {
                level,
                num_leaves: new_size - 1,
                hash: so_far,
            });
        }
    }

    pub fn root(&self) -> B256 {
        if self.size == 0 {
            return B256::ZERO;
        }

        let mut hash_so_far: Option<B256> = None;
        let mut capacity_in_hash = 0u64;
        let mut capacity = 1u64;

        for level in 0..calc_num_partials(self.size) {
            let partial = self.get_partial(level);
            if partial != B256::ZERO {
                if let Some(ref mut current) = hash_so_far {
                    while capacity_in_hash < capacity {
                        let mut combined = Vec::with_capacity(64);
                        combined.extend_from_slice(current.as_slice());
                        combined.extend_from_slice(&[0u8; 32]);
                        *current = keccak256(&combined);
                        capacity_in_hash *= 2;
                    }

                    let mut combined = Vec::with_capacity(64);
                    combined.extend_from_slice(partial.as_slice());
                    combined.extend_from_slice(current.as_slice());
                    *current = keccak256(&combined);
                    capacity_in_hash = 2 * capacity;
                } else {
                    hash_so_far = Some(partial);
                    capacity_in_hash = capacity;
                }
            }
            capacity *= 2;
        }

        hash_so_far.unwrap_or(B256::ZERO)
    }

    pub fn partials(&self) -> &[B256] {
        &self.partials
    }
}

impl Default for InMemoryMerkleAccumulator {
    fn default() -> Self {
        Self::new()
    }
}
