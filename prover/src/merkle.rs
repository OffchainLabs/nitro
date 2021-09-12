use crate::utils::Bytes32;
use digest::Digest;
use sha3::Keccak256;
use std::convert::TryFrom;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Merkle {
    prefix: &'static str,
    layers: Vec<Vec<Bytes32>>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum MerkleType {
    Value,
    Function,
    Memory,
    Table,
    TableElement,
}

fn hash_node(prefix: &str, a: Bytes32, b: Bytes32) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(prefix);
    h.update(a);
    h.update(b);
    h.finalize().into()
}

impl Merkle {
    pub fn new(ty: MerkleType, hashes: Vec<Bytes32>) -> Merkle {
        if hashes.is_empty() {
            return Merkle::default();
        }
        let prefix = match ty {
            MerkleType::Value => "Value merkle tree:",
            MerkleType::Function => "Function merkle tree:",
            MerkleType::Memory => "Memory merkle tree:",
            MerkleType::Table => "Table merkle tree:",
            MerkleType::TableElement => "Table element merkle tree:",
        };
        let mut layers = Vec::new();
        layers.push(hashes);
        while layers.last().unwrap().len() > 1 {
            let mut new_layer = Vec::new();
            for window in layers.last().unwrap().chunks(2) {
                new_layer.push(hash_node(
                    prefix,
                    window[0],
                    window.get(1).cloned().unwrap_or_default(),
                ));
            }
            layers.push(new_layer);
        }
        Merkle { prefix, layers }
    }

    pub fn root(&self) -> Bytes32 {
        if let Some(layer) = self.layers.last() {
            assert_eq!(layer.len(), 1);
            layer[0]
        } else {
            Bytes32::default()
        }
    }

    pub fn leaves(&self) -> &[Bytes32] {
        if self.layers.is_empty() {
            &[]
        } else {
            &self.layers[0]
        }
    }

    #[must_use]
    pub fn prove(&self, mut idx: usize) -> Option<Vec<u8>> {
        if idx >= self.leaves().len() {
            return None;
        }
        let mut proof = Vec::new();
        proof.push(u8::try_from(self.layers.len() - 1).unwrap());
        for layer in &self.layers {
            if layer.len() <= 1 {
                break;
            }
            let counterpart = idx ^ 1;
            proof.extend(layer.get(counterpart).cloned().unwrap_or_default());
            idx >>= 1;
        }
        Some(proof)
    }

    pub fn set(&mut self, mut idx: usize, hash: Bytes32) {
        let mut next_hash = hash;
        for layer in &mut self.layers {
            layer[idx] = next_hash;
            if layer.len() == 1 {
                // next_hash isn't needed
                break;
            }
            let counterpart = layer.get(idx ^ 1).cloned().unwrap_or_default();
            if idx % 2 == 0 {
                next_hash = hash_node(self.prefix, next_hash, counterpart);
            } else {
                next_hash = hash_node(self.prefix, counterpart, next_hash);
            }
            idx >>= 1;
        }
    }
}
