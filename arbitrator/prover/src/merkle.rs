// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use digest::Digest;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::convert::TryFrom;

#[cfg(feature = "rayon")]
use rayon::prelude::*;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum MerkleType {
    Empty,
    Value,
    Function,
    Instruction,
    Memory,
    Table,
    TableElement,
    Module,
}

impl Default for MerkleType {
    fn default() -> Self {
        Self::Empty
    }
}

impl MerkleType {
    pub fn get_prefix(self) -> &'static str {
        match self {
            MerkleType::Empty => panic!("Attempted to get prefix of empty merkle type"),
            MerkleType::Value => "Value merkle tree:",
            MerkleType::Function => "Function merkle tree:",
            MerkleType::Instruction => "Instruction merkle tree:",
            MerkleType::Memory => "Memory merkle tree:",
            MerkleType::Table => "Table merkle tree:",
            MerkleType::TableElement => "Table element merkle tree:",
            MerkleType::Module => "Module merkle tree:",
        }
    }
}

/// A Merkle tree with a fixed number of layers
/// 
/// https://en.wikipedia.org/wiki/Merkle_tree
/// 
/// Each instance's leaves contain the hashes of a specific [MerkleType].
/// The tree does not grow. It can be over-provisioned using the
/// [Merkle::new_advanced] method and passing a minimum depth.
/// 
/// This structure does not contain the data itself, only the hashes.
#[derive(Debug, Clone, PartialEq, Eq, Default, Serialize, Deserialize)]
pub struct Merkle {
    ty: MerkleType,
    layers: Vec<Vec<Bytes32>>,
    empty_layers: Vec<Bytes32>,
    min_depth: usize,
}

fn hash_node(ty: MerkleType, a: Bytes32, b: Bytes32) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(ty.get_prefix());
    h.update(a);
    h.update(b);
    h.finalize().into()
}

impl Merkle {
    // Creates a new Merkle tree with the given type and leaf hashes.
    // The tree is built up to the minimum depth necessary to hold all the leaves.
    pub fn new(ty: MerkleType, hashes: Vec<Bytes32>) -> Merkle {
        Self::new_advanced(ty, hashes, Bytes32::default(), 0)
    }

    pub fn new_advanced(
        ty: MerkleType,
        hashes: Vec<Bytes32>,
        empty_hash: Bytes32,
        min_depth: usize,
    ) -> Merkle {
        if hashes.is_empty() {
            return Merkle::default();
        }
        let mut layers = vec![hashes];
        let mut empty_layers = vec![empty_hash];
        while layers.last().unwrap().len() > 1 || layers.len() < min_depth {
            let empty_layer = *empty_layers.last().unwrap();

            #[cfg(feature = "rayon")]
            let new_layer = layers.last().unwrap().par_chunks(2);

            #[cfg(not(feature = "rayon"))]
            let new_layer = layers.last().unwrap().chunks(2);

            let new_layer = new_layer
                .map(|chunk| hash_node(ty, chunk[0], chunk.get(1).cloned().unwrap_or(empty_layer)))
                .collect();
            empty_layers.push(hash_node(ty, empty_layer, empty_layer));
            layers.push(new_layer);
        }
        Merkle {
            ty,
            layers,
            empty_layers,
            min_depth,
        }
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
    pub fn prove(&self, idx: usize) -> Option<Vec<u8>> {
        if idx >= self.leaves().len() {
            return None;
        }
        Some(self.prove_any(idx))
    }

    /// creates a merkle proof regardless of if the leaf has content
    #[must_use]
    pub fn prove_any(&self, mut idx: usize) -> Vec<u8> {
        let mut proof = vec![u8::try_from(self.layers.len() - 1).unwrap()];
        for (layer_i, layer) in self.layers.iter().enumerate() {
            if layer_i == self.layers.len() - 1 {
                break;
            }
            let counterpart = idx ^ 1;
            proof.extend(
                layer
                    .get(counterpart)
                    .cloned()
                    .unwrap_or_else(|| self.empty_layers[layer_i]),
            );
            idx >>= 1;
        }
        proof
    }

    /// Adds a new leaf to the merkle
    /// Currently O(n) in the number of leaves (could be log(n))
    pub fn push_leaf(&mut self, leaf: Bytes32) {
        let mut leaves = self.layers.swap_remove(0);
        leaves.push(leaf);
        let empty = self.empty_layers[0];
        *self = Self::new_advanced(self.ty, leaves, empty, self.min_depth);
    }

    /// Removes the rightmost leaf from the merkle
    /// Currently O(n) in the number of leaves (could be log(n))
    pub fn pop_leaf(&mut self) {
        let mut leaves = self.layers.swap_remove(0);
        leaves.pop();
        let empty = self.empty_layers[0];
        *self = Self::new_advanced(self.ty, leaves, empty, self.min_depth);
    }

    // Sets the leaf at the given index to the given hash.
    // Panics if the index is out of bounds (since the structure doesn't grow).
    pub fn set(&mut self, mut idx: usize, hash: Bytes32) {
        if self.layers[0][idx] == hash {
            return;
        }
        let mut next_hash = hash;
        let empty_layers = &self.empty_layers;
        let layers_len = self.layers.len();
        for (layer_i, layer) in self.layers.iter_mut().enumerate() {
            layer[idx] = next_hash;
            if layer_i == layers_len - 1 {
                // next_hash isn't needed
                break;
            }
            let counterpart = layer
                .get(idx ^ 1)
                .cloned()
                .unwrap_or_else(|| empty_layers[layer_i]);
            if idx % 2 == 0 {
                next_hash = hash_node(self.ty, next_hash, counterpart);
            } else {
                next_hash = hash_node(self.ty, counterpart, next_hash);
            }
            idx >>= 1;
        }
    }
}

#[test]
#[ignore]
fn simple_merkle() {
    let hashes = vec![
        Bytes32::from([1; 32]),
        Bytes32::from([2; 32]),
        Bytes32::from([3; 32]),
        Bytes32::from([4; 32]),
        Bytes32::from([5; 32]),
    ];
    let mut expected = hash_node(MerkleType::Value,
        hash_node(
            MerkleType::Value,
            hash_node(MerkleType::Value, Bytes32::from([1; 32]), Bytes32::from([2; 32])),
            hash_node(MerkleType::Value, Bytes32::from([3; 32]), Bytes32::from([4; 32]))),
        hash_node(
            MerkleType::Value,
            hash_node(MerkleType::Value, Bytes32::from([5; 32]), Bytes32::from([0; 32])),
            hash_node(MerkleType::Value, Bytes32::from([0; 32]), Bytes32::from([0; 32]))));
    let mut merkle = Merkle::new(MerkleType::Value, hashes.clone());
    assert_eq!(merkle.root(), expected);

    merkle.set(5, Bytes32::from([6; 32]));
    expected = hash_node(MerkleType::Value,
        hash_node(
            MerkleType::Value,
            hash_node(MerkleType::Value, Bytes32::from([1; 32]), Bytes32::from([2; 32])),
            hash_node(MerkleType::Value, Bytes32::from([3; 32]), Bytes32::from([4; 32]))),
        hash_node(
            MerkleType::Value,
            hash_node(MerkleType::Value, Bytes32::from([5; 32]), Bytes32::from([6; 32])),
            hash_node(MerkleType::Value, Bytes32::from([0; 32]), Bytes32::from([0; 32]))));
    assert_eq!(merkle.root(), expected);
}

#[test]
#[should_panic]
fn set_with_bad_index_panics() {
    let mut merkle = Merkle::new(MerkleType::Value, vec![Bytes32::default()]);
    merkle.set(1, Bytes32::default());
}
