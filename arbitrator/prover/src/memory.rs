// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    merkle::{Merkle, MerkleType},
    utils::Bytes32,
    value::{ArbValueType, Value},
};
use digest::Digest;
use rayon::prelude::*;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::{borrow::Cow, convert::TryFrom};

#[derive(PartialEq, Eq, Clone, Debug, Default, Serialize, Deserialize)]
pub struct Memory {
    buffer: Vec<u8>,
    #[serde(skip)]
    pub merkle: Option<Merkle>,
    pub max_size: u64,
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
        }
    }

    pub fn size(&self) -> u64 {
        self.buffer.len() as u64
    }

    pub fn merkelize(&self) -> Cow<'_, Merkle> {
        if let Some(m) = &self.merkle {
            return Cow::Borrowed(m);
        }
        // Round the size up to 8 byte long leaves, then round up to the next power of two number of leaves
        let leaves = round_up_to_power_of_two(div_round_up(self.buffer.len(), Self::LEAF_SIZE));
        let mut leaf_hashes: Vec<Bytes32> = self
            .buffer
            .par_chunks(Self::LEAF_SIZE)
            .map(|leaf| {
                let mut full_leaf = [0u8; 32];
                full_leaf[..leaf.len()].copy_from_slice(leaf);
                hash_leaf(full_leaf)
            })
            .collect();
        if leaf_hashes.len() < leaves {
            let empty_hash = hash_leaf([0u8; 32]);
            leaf_hashes.resize(leaves, empty_hash);
        }
        Cow::Owned(Merkle::new_advanced(
            MerkleType::Memory,
            leaf_hashes,
            hash_leaf([0u8; 32]),
            Self::MEMORY_LAYERS,
        ))
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
                Value::F32(f32::from_bits(contents as u32))
            }
            ArbValueType::F64 => {
                assert!(bytes == 8 && !signed, "Invalid source for f64");
                Value::F64(f64::from_bits(contents as u64))
            }
            _ => panic!("Invalid memory load output type {:?}", ty),
        })
    }

    #[must_use]
    pub fn store_value(&mut self, idx: u64, value: u64, bytes: u8) -> bool {
        let end_idx = match idx.checked_add(bytes.into()) {
            Some(x) => x,
            None => return false,
        };
        if end_idx > self.buffer.len() as u64 {
            return false;
        }
        let idx = idx as usize;
        let end_idx = end_idx as usize;
        let buf = value.to_le_bytes();
        self.buffer[idx..end_idx].copy_from_slice(&buf[..bytes.into()]);

        if let Some(mut merkle) = self.merkle.take() {
            let start_leaf = idx / Self::LEAF_SIZE;
            merkle.set(start_leaf, hash_leaf(self.get_leaf_data(start_leaf)));
            let end_leaf = (end_idx - 1) / Self::LEAF_SIZE;
            if end_leaf != start_leaf {
                merkle.set(end_leaf, hash_leaf(self.get_leaf_data(end_leaf)));
            }
            self.merkle = Some(merkle);
        }

        true
    }

    #[must_use]
    pub fn store_slice_aligned(&mut self, idx: u64, value: &[u8]) -> bool {
        if idx % Self::LEAF_SIZE as u64 != 0 {
            return false;
        }
        let end_idx = match idx.checked_add(value.len() as u64) {
            Some(x) => x,
            None => return false,
        };
        if end_idx > self.buffer.len() as u64 {
            return false;
        }
        let idx = idx as usize;
        let end_idx = end_idx as usize;
        self.buffer[idx..end_idx].copy_from_slice(value);

        if let Some(mut merkle) = self.merkle.take() {
            let start_leaf = idx / Self::LEAF_SIZE;
            merkle.set(start_leaf, hash_leaf(self.get_leaf_data(start_leaf)));
            // No need for second merkle
            assert!(value.len() <= Self::LEAF_SIZE);
        }

        true
    }

    #[must_use]
    pub fn load_32_byte_aligned(&self, idx: u64) -> Option<Bytes32> {
        if idx % Self::LEAF_SIZE as u64 != 0 {
            return None;
        }
        let idx = match usize::try_from(idx) {
            Ok(x) => x,
            Err(_) => return None,
        };

        let slice = self.get_range(idx, 32)?;
        let mut bytes = Bytes32::default();
        bytes.copy_from_slice(slice);
        Some(bytes)
    }

    pub fn get_range(&self, offset: usize, len: usize) -> Option<&[u8]> {
        let end = offset.checked_add(len)?;
        if end > self.buffer.len() {
            return None;
        }
        Some(&self.buffer[offset..end])
    }

    pub fn set_range(&mut self, offset: usize, data: &[u8]) {
        self.merkle = None;
        let end = offset
            .checked_add(data.len())
            .expect("Overflow in offset+data.len() in Memory::set_range");
        self.buffer[offset..end].copy_from_slice(data);
    }

    pub fn cache_merkle_tree(&mut self) {
        self.merkle = Some(self.merkelize().into_owned());
    }

    pub fn resize(&mut self, new_size: usize) {
        let had_merkle_tree = self.merkle.is_some();
        self.merkle = None;
        self.buffer.resize(new_size, 0);
        if had_merkle_tree {
            self.cache_merkle_tree();
        }
    }
}

#[cfg(test)]
mod test {
    use crate::memory::round_up_to_power_of_two;

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
