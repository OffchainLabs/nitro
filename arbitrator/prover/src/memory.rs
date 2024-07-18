// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    merkle::{Merkle, MerkleType},
    value::{ArbValueType, Value},
};
use arbutil::Bytes32;
use digest::Digest;
use eyre::{bail, ErrReport, Result};
use parking_lot::Mutex;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::{borrow::Cow, collections::HashSet, convert::TryFrom};

#[cfg(feature = "counters")]
use std::sync::atomic::{AtomicUsize, Ordering};

use wasmer_types::Pages;

#[cfg(feature = "rayon")]
use rayon::prelude::*;

#[cfg(feature = "counters")]
static MEM_HASH_COUNTER: AtomicUsize = AtomicUsize::new(0);

#[cfg(feature = "counters")]
pub fn reset_counters() {
    MEM_HASH_COUNTER.store(0, Ordering::Relaxed);
}

#[cfg(feature = "counters")]
pub fn print_counters() {
    println!(
        "Memory hash count: {}",
        MEM_HASH_COUNTER.load(Ordering::Relaxed)
    );
}

pub struct MemoryType {
    pub min: Pages,
    pub max: Option<Pages>,
}

impl MemoryType {
    pub fn new(min: Pages, max: Option<Pages>) -> Self {
        Self { min, max }
    }
}

impl From<&wasmer_types::MemoryType> for MemoryType {
    fn from(value: &wasmer_types::MemoryType) -> Self {
        Self::new(value.minimum, value.maximum)
    }
}

impl TryFrom<&wasmparser::MemoryType> for MemoryType {
    type Error = ErrReport;

    fn try_from(value: &wasmparser::MemoryType) -> std::result::Result<Self, Self::Error> {
        Ok(Self {
            min: Pages(value.initial.try_into()?),
            max: value.maximum.map(|x| x.try_into()).transpose()?.map(Pages),
        })
    }
}

#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Memory {
    buffer: Vec<u8>,
    #[serde(skip)]
    pub merkle: Option<Merkle>,
    pub max_size: u64,
    #[serde(skip)]
    dirty_leaves: Mutex<HashSet<usize>>,
}

fn hash_leaf(bytes: [u8; Memory::LEAF_SIZE]) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update("Memory leaf:");
    h.update(bytes);
    h.finalize().into()
}

fn round_up_to_power_of_two(mut input: usize) -> usize {
    if input == 0 {
        return 1;
    }
    input -= 1;
    1usize
        .checked_shl(usize::BITS - input.leading_zeros())
        .expect("Can't round buffer up to power of two and fit in memory")
}

/// Overflow safe divide and round up
fn div_round_up(num: usize, denom: usize) -> usize {
    let mut res = num / denom;
    if num % denom > 0 {
        res += 1;
    }
    res
}

impl Clone for Memory {
    fn clone(&self) -> Self {
        Memory {
            buffer: self.buffer.clone(),
            merkle: self.merkle.clone(),
            max_size: self.max_size,
            dirty_leaves: Mutex::new(self.dirty_leaves.lock().clone()),
        }
    }
}

impl Memory {
    pub const LEAF_SIZE: usize = 32;
    /// Only used when initializing a memory to determine its size
    pub const PAGE_SIZE: u64 = 65536;
    /// The number of layers in the memory merkle tree
    /// 1 + log2(2^32 / LEAF_SIZE) = 1 + log2(2^(32 - log2(LEAF_SIZE))) = 1 + 32 - 5
    const MEMORY_LAYERS: usize = 1 + 32 - 5;

    pub fn new(size: usize, max_size: u64) -> Memory {
        Memory {
            buffer: vec![0u8; size],
            merkle: None,
            max_size,
            dirty_leaves: Mutex::new(HashSet::new()),
        }
    }

    pub fn size(&self) -> u64 {
        self.buffer.len() as u64
    }

    pub fn merkelize(&self) -> Cow<'_, Merkle> {
        if let Some(m) = &self.merkle {
            let mut dirt = self.dirty_leaves.lock();
            for leaf_idx in dirt.drain() {
                m.set(leaf_idx, hash_leaf(self.get_leaf_data(leaf_idx)));
            }
            return Cow::Borrowed(m);
        }
        // Round the size up to 8 byte long leaves, then round up to the next power of two number of leaves
        let leaves = round_up_to_power_of_two(div_round_up(self.buffer.len(), Self::LEAF_SIZE));

        #[cfg(feature = "rayon")]
        let leaf_hashes = self.buffer.par_chunks(Self::LEAF_SIZE);

        #[cfg(not(feature = "rayon"))]
        let leaf_hashes = self.buffer.chunks(Self::LEAF_SIZE);

        let leaf_hashes: Vec<Bytes32> = leaf_hashes
            .map(|leaf| {
                let mut full_leaf = [0u8; 32];
                full_leaf[..leaf.len()].copy_from_slice(leaf);
                hash_leaf(full_leaf)
            })
            .collect();
        let size = leaf_hashes.len();
        let m = Merkle::new_advanced(MerkleType::Memory, leaf_hashes, Self::MEMORY_LAYERS);
        if size < leaves {
            m.resize(leaves).unwrap_or_else(|_| {
                panic!("Couldn't resize merkle tree from {} to {}", size, leaves)
            });
        }
        self.dirty_leaves.lock().clear();
        Cow::Owned(m)
    }

    pub fn get_leaf_data(&self, leaf_idx: usize) -> [u8; Self::LEAF_SIZE] {
        let mut buf = [0u8; Self::LEAF_SIZE];
        let idx = match leaf_idx.checked_mul(Self::LEAF_SIZE) {
            Some(x) if x < self.buffer.len() => x,
            _ => return buf,
        };
        let size = std::cmp::min(Self::LEAF_SIZE, self.buffer.len() - idx);
        buf[..size].copy_from_slice(&self.buffer[idx..(idx + size)]);
        buf
    }

    pub fn hash(&self) -> Bytes32 {
        #[cfg(feature = "counters")]
        MEM_HASH_COUNTER.fetch_add(1, Ordering::Relaxed);
        let mut h = Keccak256::new();
        h.update("Memory:");
        h.update((self.buffer.len() as u64).to_be_bytes());
        h.update(self.max_size.to_be_bytes());
        h.update(self.merkelize().root());
        h.finalize().into()
    }

    pub fn get_u8(&self, idx: u64) -> Option<u8> {
        if idx >= self.buffer.len() as u64 {
            None
        } else {
            Some(self.buffer[idx as usize])
        }
    }

    pub fn get_u16(&self, idx: u64) -> Option<u16> {
        // The index after the last index containing the u16
        let end_idx = idx.checked_add(2)?;
        if end_idx > self.buffer.len() as u64 {
            None
        } else {
            let mut buf = [0u8; 2];
            buf.copy_from_slice(&self.buffer[(idx as usize)..(end_idx as usize)]);
            Some(u16::from_le_bytes(buf))
        }
    }

    pub fn get_u32(&self, idx: u64) -> Option<u32> {
        let end_idx = idx.checked_add(4)?;
        if end_idx > self.buffer.len() as u64 {
            None
        } else {
            let mut buf = [0u8; 4];
            buf.copy_from_slice(&self.buffer[(idx as usize)..(end_idx as usize)]);
            Some(u32::from_le_bytes(buf))
        }
    }

    pub fn get_u64(&self, idx: u64) -> Option<u64> {
        let end_idx = idx.checked_add(8)?;
        if end_idx > self.buffer.len() as u64 {
            None
        } else {
            let mut buf = [0u8; 8];
            buf.copy_from_slice(&self.buffer[(idx as usize)..(end_idx as usize)]);
            Some(u64::from_le_bytes(buf))
        }
    }

    pub fn get_value(&self, idx: u64, ty: ArbValueType, bytes: u8, signed: bool) -> Option<Value> {
        let contents = match (bytes, signed) {
            (1, false) => i64::from(self.get_u8(idx)?),
            (2, false) => i64::from(self.get_u16(idx)?),
            (4, false) => i64::from(self.get_u32(idx)?),
            (8, false) => self.get_u64(idx)? as i64,
            (1, true) => i64::from(self.get_u8(idx)? as i8),
            (2, true) => i64::from(self.get_u16(idx)? as i16),
            (4, true) => i64::from(self.get_u32(idx)? as i32),
            _ => panic!(
                "Attempted to load from memory with {} bytes and signed {}",
                bytes, signed,
            ),
        };
        Some(match ty {
            ArbValueType::I32 => Value::I32(contents as u32),
            ArbValueType::I64 => Value::I64(contents as u64),
            ArbValueType::F32 => {
                assert!(bytes == 4 && !signed, "Invalid source for f32");
                f32::from_bits(contents as u32).into()
            }
            ArbValueType::F64 => {
                assert!(bytes == 8 && !signed, "Invalid source for f64");
                f64::from_bits(contents as u64).into()
            }
            _ => panic!("Invalid memory load output type {:?}", ty),
        })
    }

    #[must_use]
    // Stores a value in memory, returns false if the value would overflow the buffer.
    //
    // bytes is the number of bytes to store. It must be <= 8.
    pub fn store_value(&mut self, idx: u64, value: u64, bytes: u8) -> bool {
        assert!(bytes <= 8);
        let Some(end_idx) = idx.checked_add(bytes.into()) else {
            return false;
        };
        if end_idx > self.buffer.len() as u64 {
            return false;
        }
        let idx = idx as usize;
        let end_idx = end_idx as usize;
        let buf = value.to_le_bytes();
        self.buffer[idx..end_idx].copy_from_slice(&buf[..bytes.into()]);
        let mut dirty_leaves = self.dirty_leaves.lock();
        dirty_leaves.insert(idx / Self::LEAF_SIZE);
        dirty_leaves.insert((end_idx - 1) / Self::LEAF_SIZE);

        true
    }

    #[must_use]
    // Stores a slice in memory, returns false if the value would overflow the buffer.
    //
    // The length of value <= 32.
    pub fn store_slice_aligned(&mut self, idx: u64, value: &[u8]) -> bool {
        assert!(value.len() <= Self::LEAF_SIZE);
        if idx % Self::LEAF_SIZE as u64 != 0 {
            return false;
        }
        let Some(end_idx) = idx.checked_add(value.len() as u64) else {
            return false;
        };
        if end_idx > self.buffer.len() as u64 {
            return false;
        }
        let idx = idx as usize;
        let end_idx = end_idx as usize;
        self.buffer[idx..end_idx].copy_from_slice(value);
        self.dirty_leaves.lock().insert(idx / Self::LEAF_SIZE);

        true
    }

    #[must_use]
    pub fn load_32_byte_aligned(&self, idx: u64) -> Option<Bytes32> {
        if idx % Self::LEAF_SIZE as u64 != 0 {
            return None;
        }
        let Ok(idx) = usize::try_from(idx) else {
            return None;
        };

        let slice = self.get_range(idx, 32)?;
        let mut bytes = Bytes32::default();
        bytes.copy_from_slice(slice);
        Some(bytes)
    }

    pub fn get_range(&self, offset: usize, len: usize) -> Option<&[u8]> {
        let end: usize = offset.checked_add(len)?;
        if end > self.buffer.len() {
            return None;
        }
        Some(&self.buffer[offset..end])
    }

    pub fn set_range(&mut self, offset: usize, data: &[u8]) -> Result<()> {
        self.merkle = None;
        let Some(end) = offset.checked_add(data.len()) else {
            bail!("Overflow in offset+data.len() in Memory::set_range")
        };
        self.buffer[offset..end].copy_from_slice(data);
        Ok(())
    }

    pub fn cache_merkle_tree(&mut self) {
        self.merkle = Some(self.merkelize().into_owned());
    }

    pub fn resize(&mut self, new_size: usize) {
        self.buffer.resize(new_size, 0);
        if let Some(merkle) = &mut self.merkle {
            merkle
                .resize(new_size / Self::LEAF_SIZE)
                .unwrap_or_else(|_| {
                    panic!(
                        "Couldn't resize merkle tree from {} to {}",
                        merkle.len(),
                        new_size
                    )
                });
        }
    }
}

pub mod testing {
    use arbutil::Bytes32;

    pub fn empty_leaf_hash() -> Bytes32 {
        let leaf = [0u8; 32];
        return super::hash_leaf(leaf);
    }
}

#[cfg(test)]
mod test {
    use core::hash;

    use arbutil::Bytes32;

    use crate::memory::round_up_to_power_of_two;
    use crate::memory::testing;

    use super::Memory;

    #[test]
    pub fn fixed_memory_hash() {
        let module_memory_hash = Bytes32::from([
            86u8, 177, 192, 60, 217, 123, 221, 153, 118, 79, 229, 122, 210, 48, 187, 104, 40, 84,
            112, 63, 137, 86, 54, 2, 56, 118, 72, 158, 242, 225, 65, 80,
        ]);
        let memory = Memory::new(65536, 1);
        assert_eq!(memory.hash(), module_memory_hash);
    }

    #[test]
    pub fn empty_leaf_hash() {
        let hash = testing::empty_leaf_hash();
        print!("Bytes32([");
        for i in 0..32 {
            print!("{}", hash[i]);
            if i < 31 {
                print!(", ");
            }
        }
        print!("]);");
    }

    #[test]
    pub fn test_round_up_power_of_two() {
        assert_eq!(round_up_to_power_of_two(0), 1);
        assert_eq!(round_up_to_power_of_two(1), 1);
        assert_eq!(round_up_to_power_of_two(2), 2);
        assert_eq!(round_up_to_power_of_two(3), 4);
        assert_eq!(round_up_to_power_of_two(4), 4);
        assert_eq!(round_up_to_power_of_two(5), 8);
        assert_eq!(round_up_to_power_of_two(6), 8);
        assert_eq!(round_up_to_power_of_two(7), 8);
        assert_eq!(round_up_to_power_of_two(8), 8);
    }
}
