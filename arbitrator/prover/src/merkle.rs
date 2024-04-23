// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use digest::Digest;
use itertools::sorted;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::{collections::HashSet, convert::{TryFrom, TryInto}, sync::{Arc, Mutex, MutexGuard}};

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
/// The tree does not grow in height, but it can be initialized with fewer
/// leaves than the number that could be contained in its layers.
/// 
/// When initialized with [Merkle::new], the tree has the minimum depth
/// necessary to hold all the leaves. (e.g. 5 leaves -> 4 layers.)
/// 
/// It can be over-provisioned using the [Merkle::new_advanced] method
/// and passing a minimum depth.
/// 
/// This structure does not contain the data itself, only the hashes.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Merkle {
    ty: MerkleType,
    #[serde(with = "arc_mutex_sedre")]
    layers: Arc<Mutex<Vec<Vec<Bytes32>>>>,
    empty_layers: Vec<Bytes32>,
    min_depth: usize,
    #[serde(skip)]
    dirty_layers: Arc<Mutex<Vec<HashSet<usize>>>>,
}

fn hash_node(ty: MerkleType, a: Bytes32, b: Bytes32) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(ty.get_prefix());
    h.update(a);
    h.update(b);
    h.finalize().into()
}

#[inline]
fn capacity(layers: &Vec<Vec<Bytes32>>) -> usize {
    let base: usize = 2;
    base.pow((layers.len() - 1).try_into().unwrap())
}

impl Merkle {
    /// Creates a new Merkle tree with the given type and leaf hashes.
    /// The tree is built up to the minimum depth necessary to hold all the
    /// leaves.
    pub fn new(ty: MerkleType, hashes: Vec<Bytes32>) -> Merkle {
        Self::new_advanced(ty, hashes, Bytes32::default(), 0)
    }

    /// Creates a new Merkle tree with the given type, leaf hashes, a hash to
    /// use for representing empty leaves, and a minimum depth.
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
        let mut dirty_indices: Vec<HashSet<usize>> = Vec::new();
        while layers.last().unwrap().len() > 1 || layers.len() < min_depth {
            let empty_layer = *empty_layers.last().unwrap();

            #[cfg(feature = "rayon")]
            let new_layer = layers.last().unwrap().par_chunks(2);

            #[cfg(not(feature = "rayon"))]
            let new_layer = layers.last().unwrap().chunks(2);

            let new_layer: Vec<Bytes32> = new_layer
                .map(|chunk| hash_node(ty, chunk[0], chunk.get(1).cloned().unwrap_or(empty_layer)))
                .collect();
            empty_layers.push(hash_node(ty, empty_layer, empty_layer));
            dirty_indices.push(HashSet::with_capacity(new_layer.len()));
            layers.push(new_layer);
        }
        let dirty_layers = Arc::new(Mutex::new(dirty_indices));
        Merkle {
            ty,
            layers: Arc::new(Mutex::new(layers)),
            empty_layers,
            min_depth,
            dirty_layers,
        }
    }

    fn rehash(&self) {
        let dirty_layers = &mut self.dirty_layers.lock().unwrap();
        if dirty_layers.is_empty() || dirty_layers[0].is_empty() {
            return;
        }
        let layers = &mut self.layers.lock().unwrap();
        for layer_i in 1..layers.len() {
            let dirty_i = layer_i - 1;
            let dirt = dirty_layers[dirty_i].clone();
            for idx in sorted(dirt.iter()) {
                let left_child_idx = idx << 1;
                let right_child_idx = left_child_idx + 1;
                let left = layers[layer_i -1][left_child_idx];
                let right = layers[layer_i-1]
                    .get(right_child_idx)
                    .cloned()
                    .unwrap_or_else(|| self.empty_layers[layer_i - 1]);
                let new_hash = hash_node(self.ty, left, right);
                if *idx < layers[layer_i].len() {
                    layers[layer_i][*idx] = new_hash;
                } else {
                    layers[layer_i].push(new_hash);
                }
                if layer_i < layers.len() - 1 {
                    dirty_layers[dirty_i + 1].insert(idx >> 1);
                }
            }
            dirty_layers[dirty_i].clear();
        }
    }

    pub fn root(&self) -> Bytes32 {
        self.rehash();
        if let Some(layer) = self.layers.lock().unwrap().last() {
            assert_eq!(layer.len(), 1);
            layer[0]
        } else {
            Bytes32::default()
        }
    }

    // Returns the total number of leaves the tree can hold.
    #[inline]
    #[cfg(test)]
    fn capacity(&self) -> usize {
        return capacity(self.layers.lock().unwrap().as_ref());
    }

    // Returns the number of leaves in the tree.
    pub fn len(&self) -> usize {
        self.layers.lock().unwrap()[0].len()
    }

    #[must_use]
    pub fn prove(&self, idx: usize) -> Option<Vec<u8>> {
        if self.layers.lock().unwrap().is_empty() || idx >= self.layers.lock().unwrap()[0].len() {
            return None;
        }
        Some(self.prove_any(idx))
    }

    /// creates a merkle proof regardless of if the leaf has content
    #[must_use]
    pub fn prove_any(&self, mut idx: usize) -> Vec<u8> {
        let layers = self.layers.lock().unwrap();
        let mut proof = vec![u8::try_from(layers.len() - 1).unwrap()];
        for (layer_i, layer) in layers.iter().enumerate() {
            if layer_i == layers.len() - 1 {
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
        let mut leaves = self.layers.lock().unwrap().swap_remove(0);
        leaves.push(leaf);
        let empty = self.empty_layers[0];
        *self = Self::new_advanced(self.ty, leaves, empty, self.min_depth);
    }

    /// Removes the rightmost leaf from the merkle
    /// Currently O(n) in the number of leaves (could be log(n))
    pub fn pop_leaf(&mut self) {
        let mut leaves = self.layers.lock().unwrap().swap_remove(0);
        leaves.pop();
        let empty = self.empty_layers[0];
        *self = Self::new_advanced(self.ty, leaves, empty, self.min_depth);
    }

    // Sets the leaf at the given index to the given hash.
    // Panics if the index is out of bounds (since the structure doesn't grow).
    pub fn set(&self, idx: usize, hash: Bytes32) {
        let mut layers = self.layers.lock().unwrap();
        self.locked_set(&mut layers, idx, hash);
    }

    fn locked_set(&self, locked_layers: &mut MutexGuard<Vec<Vec<Bytes32>>>, idx: usize, hash: Bytes32) {
        if locked_layers[0][idx] == hash {
            return;
        }
        locked_layers[0][idx] = hash;
        self.dirty_layers.lock().unwrap()[0].insert(idx >> 1);
    }

    /// Extends the leaves of the tree with the given hashes.
    /// 
    /// Returns the new number of leaves in the tree.
    /// Erorrs if the number of hashes plus the current leaves is greater than
    /// the capacity of the tree.
    pub fn extend(&self, hashes: Vec<Bytes32>) -> Result<usize, String> {
        let mut layers = self.layers.lock().unwrap();
        if hashes.len() > capacity(layers.as_ref()) - layers[0].len() {
            return Err("Cannot extend with more leaves than the capicity of the tree.".to_owned());
        }
        let mut idx = layers[0].len();
        layers[0].resize(idx + hashes.len(), self.empty_layers[0]);
        for hash in hashes {
            self.locked_set(&mut layers, idx, hash);
            idx += 1;
        }
        return Ok(layers[0].len());
    }
}

impl PartialEq for Merkle {
    fn eq(&self, other: &Self) -> bool {
        self.root() == other.root()
    }
}

impl Eq for Merkle {}

pub mod arc_mutex_sedre {
    pub fn serialize<S, T>(data: &std::sync::Arc<std::sync::Mutex<T>>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
        T: serde::Serialize,
    {
        data.lock().unwrap().serialize(serializer)
    }

    pub fn deserialize<'de, D, T>(deserializer: D) -> Result<std::sync::Arc<std::sync::Mutex<T>>, D::Error>
    where
        D: serde::Deserializer<'de>,
        T: serde::Deserialize<'de>,
    {
        Ok(std::sync::Arc::new(std::sync::Mutex::new(T::deserialize(deserializer)?)))
    }
}

#[test]
fn extend_works() {
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
    let merkle = Merkle::new(MerkleType::Value, hashes.clone());
    assert_eq!(merkle.capacity(), 8);
    assert_eq!(merkle.root(), expected);

    let new_size = match merkle.extend(vec![Bytes32::from([6; 32])]) {
        Ok(size) => size,
        Err(e) => panic!("{}", e)
    };
    assert_eq!(new_size, 6);
    expected = hash_node(MerkleType::Value,
        hash_node(
            MerkleType::Value,
            hash_node(MerkleType::Value, Bytes32::from([1; 32]), Bytes32::from([2; 32])),
            hash_node(MerkleType::Value, Bytes32::from([3; 32]), Bytes32::from([4; 32]))),
        hash_node(
            MerkleType::Value,
            hash_node(MerkleType::Value, Bytes32::from([5; 32]), Bytes32::from([6; 32])),
            hash_node(MerkleType::Value, Bytes32::from([0; 32]), Bytes32::from([0; 32]))));
    assert_eq!(merkle.capacity(), 8);
    assert_eq!(merkle.root(), expected);
    merkle.prove(1).unwrap();
}

#[test]
fn correct_capacity() {
    let merkle = Merkle::new(MerkleType::Value, vec![Bytes32::from([1; 32])]);
    assert_eq!(merkle.capacity(), 1);
    let merkle = Merkle::new_advanced(MerkleType::Memory, vec![Bytes32::from([1; 32])], Bytes32::default(), 11);
    assert_eq!(merkle.capacity(), 1024);
}

#[test]
#[should_panic(expected = "index out of bounds")]
fn set_with_bad_index_panics() {
    let merkle = Merkle::new(MerkleType::Value, vec![Bytes32::default(), Bytes32::default()]);
    assert_eq!(merkle.capacity(), 2);
    merkle.set(2, Bytes32::default());
}
