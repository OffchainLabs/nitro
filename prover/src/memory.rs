use crate::{
    merkle::{Merkle, MerkleType},
    utils::{usize_to_u256_bytes, Bytes32},
    value::{Value, ValueType},
};
use digest::Digest;
use sha3::Keccak256;

#[derive(PartialEq, Eq, Clone, Debug, Default)]
pub struct Memory {
    buffer: Vec<u8>,
}

fn hash_leaf(bytes: &[u8]) -> Bytes32 {
    let mut padded_bytes = [0u8; 32];
    padded_bytes[..bytes.len()].copy_from_slice(bytes);
    let mut h = Keccak256::new();
    h.update("Memory leaf:");
    h.update(padded_bytes);
    h.finalize().into()
}

impl Memory {
    pub fn new(size: usize) -> Memory {
        Memory {
            buffer: vec![0u8; size],
        }
    }

    pub fn size(&self) -> u64 {
        self.buffer.len() as u64
    }

    pub fn merkelize(&self) -> Merkle {
        let leafs = self.buffer.chunks(32).map(hash_leaf).collect();
        Merkle::new(MerkleType::Memory, leafs)
    }

    pub fn hash(&self) -> Bytes32 {
        let mut h = Keccak256::new();
        h.update("Memory:");
        h.update(usize_to_u256_bytes(self.buffer.len()));
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

    pub fn get_value(&self, idx: u64, ty: ValueType, bytes: u8, signed: bool) -> Option<Value> {
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
            ValueType::I32 => Value::I32(contents as u32),
            ValueType::I64 => Value::I64(contents as u64),
            ValueType::F32 => {
                assert!(bytes == 4 && !signed, "Invalid source for f32");
                Value::F32(f32::from_bits(contents as u32))
            }
            ValueType::F64 => {
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
		let buf = value.to_le_bytes();
		self.buffer[(idx as usize)..(end_idx as usize)].copy_from_slice(&buf[..bytes.into()]);
		true
	}
}
