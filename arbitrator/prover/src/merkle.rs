// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use digest::Digest;

use enum_iterator::Sequence;

#[cfg(feature = "counters")]
use enum_iterator::all;


#[cfg(feature = "counters")]
use std::sync::atomic::AtomicUsize;

#[cfg(feature = "counters")]
use std::sync::atomic::Ordering;

#[cfg(feature = "counters")]
use lazy_static::lazy_static;


#[cfg(feature = "counters")]
use std::collections::HashMap;

use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use core::panic;
use std::convert::{TryFrom, TryInto};

#[cfg(feature = "rayon")]
use rayon::prelude::*;

#[cfg(feature = "counters")]
lazy_static! {
    static ref NEW_COUNTERS: HashMap<&'static MerkleType, AtomicUsize> = {
        let mut map = HashMap::new();
        map.insert(&MerkleType::Empty, AtomicUsize::new(0));
        map.insert(&MerkleType::Value, AtomicUsize::new(0));
        map.insert(&MerkleType::Function, AtomicUsize::new(0));
        map.insert(&MerkleType::Instruction, AtomicUsize::new(0));
        map.insert(&MerkleType::Memory, AtomicUsize::new(0));
        map.insert(&MerkleType::Table, AtomicUsize::new(0));
        map.insert(&MerkleType::TableElement, AtomicUsize::new(0));
        map.insert(&MerkleType::Module, AtomicUsize::new(0));
        map
    };
}
#[cfg(feature = "counters")]
lazy_static! {
    static ref ROOT_COUNTERS: HashMap<&'static MerkleType, AtomicUsize> = {
        let mut map = HashMap::new();
        map.insert(&MerkleType::Empty, AtomicUsize::new(0));
        map.insert(&MerkleType::Value, AtomicUsize::new(0));
        map.insert(&MerkleType::Function, AtomicUsize::new(0));
        map.insert(&MerkleType::Instruction, AtomicUsize::new(0));
        map.insert(&MerkleType::Memory, AtomicUsize::new(0));
        map.insert(&MerkleType::Table, AtomicUsize::new(0));
        map.insert(&MerkleType::TableElement, AtomicUsize::new(0));
        map.insert(&MerkleType::Module, AtomicUsize::new(0));
        map
    };
}
#[cfg(feature = "counters")]
lazy_static! {
    static ref SET_COUNTERS: HashMap<&'static MerkleType, AtomicUsize> = {
        let mut map = HashMap::new();
        map.insert(&MerkleType::Empty, AtomicUsize::new(0));
        map.insert(&MerkleType::Value, AtomicUsize::new(0));
        map.insert(&MerkleType::Function, AtomicUsize::new(0));
        map.insert(&MerkleType::Instruction, AtomicUsize::new(0));
        map.insert(&MerkleType::Memory, AtomicUsize::new(0));
        map.insert(&MerkleType::Table, AtomicUsize::new(0));
        map.insert(&MerkleType::TableElement, AtomicUsize::new(0));
        map.insert(&MerkleType::Module, AtomicUsize::new(0));
        map
    };
}


#[derive(Debug, Clone, Copy, Hash, PartialEq, Eq, Serialize, Deserialize, Sequence)]
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

#[cfg(feature = "counters")]
pub fn print_counters() {
    for ty in all::<MerkleType>() {
        if ty == MerkleType::Empty {
            continue;
        }
        println!("{} New: {}, Root: {}, Set: {}",
        ty.get_prefix(), NEW_COUNTERS[&ty].load(Ordering::Relaxed), ROOT_COUNTERS[&ty].load(Ordering::Relaxed), SET_COUNTERS[&ty].load(Ordering::Relaxed));
    }
}

#[cfg(feature = "counters")]
pub fn reset_counters() {
    for ty in all::<MerkleType>() {
        if ty == MerkleType::Empty {
            continue;
        }
        NEW_COUNTERS[&ty].store(0, Ordering::Relaxed);
        ROOT_COUNTERS[&ty].store(0, Ordering::Relaxed);
        SET_COUNTERS[&ty].store(0, Ordering::Relaxed);
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
        #[cfg(feature = "counters")]
        NEW_COUNTERS[&ty].fetch_add(1, Ordering::Relaxed);
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
        #[cfg(feature = "counters")]
        ROOT_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
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

    // Returns the total number of leaves the tree can hold.
    #[inline]
    fn capacity(&self) -> usize {
        let base: usize = 2;
        base.pow((self.layers.len() -1).try_into().unwrap())
    }

    // Returns the number of leaves in the tree.
    pub fn len(&self) -> usize {
        self.layers[0].len()
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
        #[cfg(feature = "counters")]
        SET_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
        if self.layers[0][idx] == hash {
            return;
        }
        let mut next_hash = hash;
        let empty_layers = &self.empty_layers;
        let layers_len = self.layers.len();
        for (layer_i, layer) in self.layers.iter_mut().enumerate() {
            if idx < layer.len() {
                layer[idx] = next_hash;
            } else if idx == layer.len() {
                layer.push(next_hash);
            } else {
                panic!("Index {} out of bounds {}", idx, layer.len());
            }
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

    /// Extends the leaves of the tree with the given hashes.
    /// 
    /// Returns the new number of leaves in the tree.
    /// Erorrs if the number of hashes plus the current leaves is greater than
    /// the capacity of the tree.
    pub fn extend(&mut self, hashes: Vec<Bytes32>) -> Result<usize, String> {
        if hashes.len() > self.capacity() - self.layers[0].len() {
            return Err("Cannot extend with more leaves than the capicity of the tree.".to_owned());
        }
        let mut idx = self.layers[0].len();
        self.layers[0].resize(idx + hashes.len(), self.empty_layers[0]);
        for hash in hashes {
            self.set(idx, hash);
            idx += 1;
        }
        return Ok(self.layers[0].len());
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
    let mut merkle = Merkle::new(MerkleType::Value, hashes.clone());
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
    assert_eq!(merkle.root(), expected);
}

#[test]
fn correct_capacity() {
    let merkle = Merkle::new(MerkleType::Value, vec![Bytes32::from([1; 32])]);
    assert_eq!(merkle.capacity(), 1);
    let merkle = Merkle::new_advanced(MerkleType::Memory, vec![Bytes32::from([1; 32])], Bytes32::default(), 11);
    assert_eq!(merkle.capacity(), 1024);
}

#[test]
#[should_panic]
fn set_with_bad_index_panics() {
    let mut merkle = Merkle::new(MerkleType::Value, vec![Bytes32::default(), Bytes32::default()]);
    assert_eq!(merkle.capacity(), 2);
    merkle.set(2, Bytes32::default());
}
