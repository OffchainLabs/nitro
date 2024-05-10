// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use digest::Digest;

use enum_iterator::Sequence;

#[cfg(feature = "counters")]
use enum_iterator::all;
use itertools::Itertools;

use std::cmp::max;

#[cfg(feature = "counters")]
use std::sync::atomic::AtomicUsize;

#[cfg(feature = "counters")]
use std::sync::atomic::Ordering;

#[cfg(feature = "counters")]
use lazy_static::lazy_static;

#[cfg(feature = "counters")]
use std::collections::HashMap;

use core::panic;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::{
    collections::HashSet,
    convert::{TryFrom, TryInto},
    sync::{Arc, Mutex},
};

#[cfg(feature = "rayon")]
use rayon::prelude::*;

mod zerohashes;

use zerohashes::ZERO_HASHES;

use self::zerohashes::EMPTY_HASH;

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
#[cfg(feature = "counters")]
lazy_static! {
    static ref RESIZE_COUNTERS: HashMap<&'static MerkleType, AtomicUsize> = {
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
        println!(
            "{} New: {}, Root: {}, Set: {} Resize: {}",
            ty.get_prefix(),
            NEW_COUNTERS[&ty].load(Ordering::Relaxed),
            ROOT_COUNTERS[&ty].load(Ordering::Relaxed),
            SET_COUNTERS[&ty].load(Ordering::Relaxed),
            RESIZE_COUNTERS[&ty].load(Ordering::Relaxed),
        );
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
        RESIZE_COUNTERS[&ty].store(0, Ordering::Relaxed);
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
    min_depth: usize,
    #[serde(with = "arc_mutex_sedre")]
    dirty_layers: Arc<Mutex<Vec<HashSet<usize>>>>,
}

fn hash_node(ty: MerkleType, a: impl AsRef<[u8]>, b: impl AsRef<[u8]>) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(ty.get_prefix());
    h.update(a);
    h.update(b);
    h.finalize().into()
}

const fn empty_hash_at(ty: MerkleType, layer_i: usize) -> &'static Bytes32 {
    match ty {
        MerkleType::Empty => EMPTY_HASH,
        MerkleType::Value => &ZERO_HASHES[0][layer_i],
        MerkleType::Function => &ZERO_HASHES[1][layer_i],
        MerkleType::Instruction => &ZERO_HASHES[2][layer_i],
        MerkleType::Memory => &ZERO_HASHES[3][layer_i],
        MerkleType::Table => &ZERO_HASHES[4][layer_i],
        MerkleType::TableElement => &ZERO_HASHES[5][layer_i],
        MerkleType::Module => &ZERO_HASHES[6][layer_i],
    }
}

#[inline]
#[cfg(feature = "rayon")]
fn new_layer(ty: MerkleType, layer: &Vec<Bytes32>, empty_hash: &'static Bytes32) -> Vec<Bytes32> {
    let mut new_layer: Vec<Bytes32> = Vec::with_capacity(layer.len() >> 1);
    let chunks = layer.par_chunks(2);
    chunks
        .map(|chunk| hash_node(ty, chunk[0], chunk.get(1).unwrap_or(empty_hash)))
        .collect_into_vec(&mut new_layer);
    new_layer
}

#[inline]
#[cfg(not(feature = "rayon"))]
fn new_layer(ty: MerkleType, layer: &Vec<Bytes32>, empty_hash: &'static Bytes32) -> Vec<Bytes32> {
    let new_layer = layer
        .chunks(2)
        .map(|chunk| hash_node(ty, chunk[0], chunk.get(1).unwrap_or(empty_hash)))
        .collect();
    new_layer
}

impl Merkle {
    /// Creates a new Merkle tree with the given type and leaf hashes.
    /// The tree is built up to the minimum depth necessary to hold all the
    /// leaves.
    pub fn new(ty: MerkleType, hashes: Vec<Bytes32>) -> Merkle {
        Self::new_advanced(ty, hashes, 0)
    }

    /// Creates a new Merkle tree with the given type, leaf hashes, a hash to
    /// use for representing empty leaves, and a minimum depth.
    pub fn new_advanced(ty: MerkleType, hashes: Vec<Bytes32>, min_depth: usize) -> Merkle {
        #[cfg(feature = "counters")]
        NEW_COUNTERS[&ty].fetch_add(1, Ordering::Relaxed);
        if hashes.is_empty() && min_depth == 0 {
            return Merkle::default();
        }
        let mut depth = (hashes.len() as f64).log2().ceil() as usize;
        depth = depth.max(min_depth);
        let mut layers: Vec<Vec<Bytes32>> = Vec::with_capacity(depth);
        layers.push(hashes);
        let mut dirty_indices: Vec<HashSet<usize>> = Vec::with_capacity(depth);
        let mut layer_i = 0usize;
        while layers.last().unwrap().len() > 1 || layers.len() < min_depth {
            let layer = layers.last().unwrap();
            let empty_hash = empty_hash_at(ty, layer_i);

            let new_layer = new_layer(ty, layer, empty_hash);
            dirty_indices.push(HashSet::with_capacity(new_layer.len()));
            layers.push(new_layer);
            layer_i += 1;
        }
        let dirty_layers = Arc::new(Mutex::new(dirty_indices));
        Merkle {
            ty,
            layers: Arc::new(Mutex::new(layers)),
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
            for idx in dirt.iter().sorted() {
                let left_child_idx = idx << 1;
                let right_child_idx = left_child_idx + 1;
                let left = layers[layer_i - 1][left_child_idx];
                let right = layers[layer_i - 1]
                    .get(right_child_idx)
                    .unwrap_or(empty_hash_at(self.ty, layer_i - 1));
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
        #[cfg(feature = "counters")]
        ROOT_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
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
    fn capacity(&self) -> usize {
        let layers = self.layers.lock().unwrap();
        if layers.is_empty() {
            return 0;
        }
        let base: usize = 2;
        base.pow((layers.len() - 1).try_into().unwrap())
    }

    // Returns the number of leaves in the tree.
    pub fn len(&self) -> usize {
        self.layers.lock().unwrap()[0].len()
    }

    pub fn is_empty(&self) -> bool {
        let layers = self.layers.lock().unwrap();
        layers.is_empty() || layers[0].is_empty()
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
        self.rehash();
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
                    .unwrap_or_else(|| *empty_hash_at(self.ty, layer_i)),
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
        *self = Self::new_advanced(self.ty, leaves, self.min_depth);
    }

    /// Removes the rightmost leaf from the merkle
    /// Currently O(n) in the number of leaves (could be log(n))
    pub fn pop_leaf(&mut self) {
        let mut leaves = self.layers.lock().unwrap().swap_remove(0);
        leaves.pop();
        *self = Self::new_advanced(self.ty, leaves, self.min_depth);
    }

    // Sets the leaf at the given index to the given hash.
    // Panics if the index is out of bounds (since the structure doesn't grow).
    pub fn set(&self, idx: usize, hash: Bytes32) {
        #[cfg(feature = "counters")]
        SET_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
        let mut layers = self.layers.lock().unwrap();
        if layers[0][idx] == hash {
            return;
        }
        layers[0][idx] = hash;
        self.dirty_layers.lock().unwrap()[0].insert(idx >> 1);
    }

    /// Resizes the number of leaves the tree can hold.
    ///
    /// The extra space is filled with empty hashes.
    pub fn resize(&self, new_len: usize) -> Result<usize, String> {
        #[cfg(feature = "counters")]
        RESIZE_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
        if new_len > self.capacity() {
            return Err(
                "Cannot resize to a length greater than the capacity of the tree.".to_owned(),
            );
        }
        let mut layers = self.layers.lock().unwrap();
        let mut layer_size = new_len;
        for (layer_i, layer) in layers.iter_mut().enumerate() {
            layer.resize(layer_size, *empty_hash_at(self.ty, layer_i));
            layer_size = max(layer_size >> 1, 1);
        }
        let start = layers[0].len();
        for i in start..new_len {
            self.dirty_layers.lock().unwrap()[0].insert(i);
        }
        Ok(layers[0].len())
    }
}

impl PartialEq for Merkle {
    fn eq(&self, other: &Self) -> bool {
        self.root() == other.root()
    }
}

impl Eq for Merkle {}

pub mod arc_mutex_sedre {
    pub fn serialize<S, T>(
        data: &std::sync::Arc<std::sync::Mutex<T>>,
        serializer: S,
    ) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
        T: serde::Serialize,
    {
        data.lock().unwrap().serialize(serializer)
    }

    pub fn deserialize<'de, D, T>(
        deserializer: D,
    ) -> Result<std::sync::Arc<std::sync::Mutex<T>>, D::Error>
    where
        D: serde::Deserializer<'de>,
        T: serde::Deserialize<'de>,
    {
        Ok(std::sync::Arc::new(std::sync::Mutex::new(T::deserialize(
            deserializer,
        )?)))
    }
}

#[test]
fn resize_works() {
    let hashes = vec![
        Bytes32::from([1; 32]),
        Bytes32::from([2; 32]),
        Bytes32::from([3; 32]),
        Bytes32::from([4; 32]),
        Bytes32::from([5; 32]),
    ];
    let mut expected = hash_node(
        MerkleType::Value,
        hash_node(
            MerkleType::Value,
            hash_node(
                MerkleType::Value,
                Bytes32::from([1; 32]),
                Bytes32::from([2; 32]),
            ),
            hash_node(
                MerkleType::Value,
                Bytes32::from([3; 32]),
                Bytes32::from([4; 32]),
            ),
        ),
        hash_node(
            MerkleType::Value,
            hash_node(
                MerkleType::Value,
                Bytes32::from([5; 32]),
                Bytes32::from([0; 32]),
            ),
            hash_node(
                MerkleType::Value,
                Bytes32::from([0; 32]),
                Bytes32::from([0; 32]),
            ),
        ),
    );
    let merkle = Merkle::new(MerkleType::Value, hashes.clone());
    assert_eq!(merkle.capacity(), 8);
    assert_eq!(merkle.root(), expected);

    let new_size = match merkle.resize(6) {
        Ok(size) => size,
        Err(e) => panic!("{}", e),
    };
    assert_eq!(new_size, 6);
    merkle.set(5, Bytes32::from([6; 32]));
    expected = hash_node(
        MerkleType::Value,
        hash_node(
            MerkleType::Value,
            hash_node(
                MerkleType::Value,
                Bytes32::from([1; 32]),
                Bytes32::from([2; 32]),
            ),
            hash_node(
                MerkleType::Value,
                Bytes32::from([3; 32]),
                Bytes32::from([4; 32]),
            ),
        ),
        hash_node(
            MerkleType::Value,
            hash_node(
                MerkleType::Value,
                Bytes32::from([5; 32]),
                Bytes32::from([6; 32]),
            ),
            hash_node(
                MerkleType::Value,
                Bytes32::from([0; 32]),
                Bytes32::from([0; 32]),
            ),
        ),
    );
    assert_eq!(merkle.root(), expected);
}

#[test]
fn correct_capacity() {
    let merkle = Merkle::new(MerkleType::Value, vec![Bytes32::from([1; 32])]);
    assert_eq!(merkle.capacity(), 1);
    let merkle = Merkle::new_advanced(MerkleType::Memory, vec![Bytes32::from([1; 32])], 11);
    assert_eq!(merkle.capacity(), 1024);
}

#[test]
#[ignore = "This is just used for generating the zero hashes for the memory merkle trees."]
fn emit_memory_zerohashes() {
    // The following code was generated from the empty_leaf_hash() test in the memory package.
    let mut empty_node = Bytes32::new_direct([
        57, 29, 211, 154, 252, 227, 18, 99, 65, 126, 203, 166, 252, 232, 32, 3, 98, 194, 254, 186,
        118, 14, 139, 192, 101, 156, 55, 194, 101, 11, 11, 168,
    ])
    .clone();
    for _ in 0..64 {
        print!("Bytes32::new_direct([");
        for i in 0..32 {
            print!("{}", empty_node[i]);
            if i < 31 {
                print!(", ");
            }
        }
        println!("]),");
        empty_node = hash_node(MerkleType::Memory, empty_node, empty_node);
    }
}

#[test]
fn serialization_roundtrip() {
    let merkle = Merkle::new_advanced(MerkleType::Value, vec![Bytes32::from([1; 32])], 4);
    merkle.resize(4).expect("resize failed");
    merkle.set(3, Bytes32::from([2; 32]));
    let serialized = bincode::serialize(&merkle).unwrap();
    let deserialized: Merkle = bincode::deserialize(&serialized).unwrap();
    assert_eq!(merkle, deserialized);
}

#[test]
#[should_panic(expected = "index out of bounds")]
fn set_with_bad_index_panics() {
    let merkle = Merkle::new(
        MerkleType::Value,
        vec![Bytes32::default(), Bytes32::default()],
    );
    assert_eq!(merkle.capacity(), 2);
    merkle.set(2, Bytes32::default());
}
