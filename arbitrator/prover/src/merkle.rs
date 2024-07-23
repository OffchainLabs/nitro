// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use bitvec::prelude::*;
use core::panic;
use digest::Digest;
use enum_iterator::Sequence;
use parking_lot::Mutex;
use serde::{Deserialize, Serialize};
use sha3::Keccak256;
use std::cmp::max;
use std::convert::{TryFrom, TryInto};

#[cfg(feature = "rayon")]
use rayon::prelude::*;

mod zerohashes;
use self::zerohashes::{EMPTY_HASH, ZERO_HASHES};
#[cfg(feature = "counters")]
use {
    enum_iterator::all,
    lazy_static::lazy_static,
    std::collections::HashMap,
    std::sync::atomic::{AtomicUsize, Ordering},
};

#[cfg(feature = "counters")]
fn create_counters_hashmap() -> HashMap<MerkleType, AtomicUsize> {
    let mut map = HashMap::new();
    for ty in all::<MerkleType>() {
        map.insert(ty, AtomicUsize::new(0));
    }
    map
}

#[cfg(feature = "counters")]
lazy_static! {
    static ref NEW_COUNTERS: HashMap<MerkleType, AtomicUsize> = create_counters_hashmap();
    static ref ROOT_COUNTERS: HashMap<MerkleType, AtomicUsize> = create_counters_hashmap();
    static ref SET_COUNTERS: HashMap<MerkleType, AtomicUsize> = create_counters_hashmap();
    static ref RESIZE_COUNTERS: HashMap<MerkleType, AtomicUsize> = create_counters_hashmap();
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

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
struct Layers {
    data: Vec<Vec<Bytes32>>,
    dirty_leaf_parents: BitVec,
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
#[derive(Debug, Default, Serialize, Deserialize)]
pub struct Merkle {
    ty: MerkleType,
    #[serde(with = "mutex_sedre")]
    layers: Mutex<Layers>,
    min_depth: usize,
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
fn new_layer(ty: MerkleType, layer: &[Bytes32], empty_hash: &'static Bytes32) -> Vec<Bytes32> {
    let mut new_layer: Vec<Bytes32> = Vec::with_capacity(layer.len() >> 1);
    let chunks = layer.par_chunks(2);
    chunks
        .map(|chunk| hash_node(ty, chunk[0], chunk.get(1).unwrap_or(empty_hash)))
        .collect_into_vec(&mut new_layer);
    new_layer
}

#[inline]
#[cfg(not(feature = "rayon"))]
fn new_layer(ty: MerkleType, layer: &[Bytes32], empty_hash: &'static Bytes32) -> Vec<Bytes32> {
    let new_layer = layer
        .chunks(2)
        .map(|chunk| hash_node(ty, chunk[0], chunk.get(1).unwrap_or(empty_hash)))
        .collect();
    new_layer
}

impl Clone for Merkle {
    fn clone(&self) -> Self {
        let leaves = self.layers.lock().data.first().cloned().unwrap_or_default();
        Merkle::new_advanced(self.ty, leaves, self.min_depth)
    }
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
        let depth = if hashes.len() > 1 {
            min_depth.max(((hashes.len() - 1).ilog2() + 1).try_into().unwrap())
        } else {
            min_depth
        };
        let mut layers: Vec<Vec<Bytes32>> = Vec::with_capacity(depth);
        let dirty_leaf_parents = bitvec![0; hashes.len() + 1 >> 1];
        layers.push(hashes);
        while layers.last().unwrap().len() > 1 || layers.len() < min_depth {
            let layer = layers.last().unwrap();
            let empty_hash = empty_hash_at(ty, layers.len() - 1);

            let new_layer = new_layer(ty, layer, empty_hash);
            layers.push(new_layer);
        }
        let layers = Mutex::new(Layers {
            data: layers,
            dirty_leaf_parents,
        });
        Merkle {
            ty,
            layers,
            min_depth,
        }
    }

    fn rehash(&self, layers: &mut Layers) {
        // If nothing is dirty, then there's no need to rehash.
        if layers.dirty_leaf_parents.is_empty() {
            return;
        }
        // Replace the dirty leaf parents with clean ones.
        let mut dirt = std::mem::replace(
            &mut layers.dirty_leaf_parents,
            bitvec![0; (layers.data[0].len() + 1) >> 1],
        );
        // Process dirty indices starting from layer 1 (layer 0 is the leaves).
        for layer_i in 1..layers.data.len() {
            let mut new_dirt = bitvec![0; dirt.len() + 1 >> 1];
            for idx in dirt.iter_ones() {
                let left_child_idx = idx << 1;
                let right_child_idx = left_child_idx + 1;
                // The left child is guaranteed to exist, but the right one
                // might not if the number of child nodes is odd.
                let left = layers.data[layer_i - 1][left_child_idx];
                let right = layers.data[layer_i - 1]
                    .get(right_child_idx)
                    .unwrap_or(empty_hash_at(self.ty, layer_i - 1));
                let new_hash = hash_node(self.ty, left, right);
                if idx < layers.data[layer_i].len() {
                    layers.data[layer_i][idx] = new_hash;
                } else {
                    // Push the new parent hash onto the end of the layer.
                    assert_eq!(idx, layers.data[layer_i].len());
                    layers.data[layer_i].push(new_hash);
                }
                // Mark the node's parent as dirty unless it's the root.
                if layer_i < layers.data.len() - 1 {
                    new_dirt.set(idx >> 1, true);
                }
            }
            dirt = new_dirt;
        }
    }

    pub fn root(&self) -> Bytes32 {
        #[cfg(feature = "counters")]
        ROOT_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
        let mut layers = self.layers.lock();
        self.rehash(&mut layers);
        if let Some(layer) = layers.data.last() {
            assert_eq!(layer.len(), 1);
            layer[0]
        } else {
            Bytes32::default()
        }
    }

    // Returns the total number of leaves the tree can hold.
    #[inline]
    fn capacity(&self) -> usize {
        let layers = self.layers.lock();
        if layers.data.is_empty() {
            return 0;
        }
        1 << (layers.data.len() - 1)
    }

    // Returns the number of leaves in the tree.
    pub fn len(&self) -> usize {
        self.layers.lock().data[0].len()
    }

    pub fn is_empty(&self) -> bool {
        let layers = self.layers.lock();
        layers.data.is_empty() || layers.data[0].is_empty()
    }

    #[must_use]
    pub fn prove(&self, idx: usize) -> Option<Vec<u8>> {
        {
            let layers = self.layers.lock();
            if layers.data.is_empty() || idx >= layers.data[0].len() {
                return None;
            }
        }
        Some(self.prove_any(idx))
    }

    /// creates a merkle proof regardless of if the leaf has content
    #[must_use]
    pub fn prove_any(&self, mut idx: usize) -> Vec<u8> {
        let mut layers = self.layers.lock();
        self.rehash(&mut layers);
        let mut proof = vec![u8::try_from(layers.data.len() - 1).unwrap()];
        for (layer_i, layer) in layers.data.iter().enumerate() {
            if layer_i == layers.data.len() - 1 {
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
        let mut leaves = self.layers.lock().data.swap_remove(0);
        leaves.push(leaf);
        *self = Self::new_advanced(self.ty, leaves, self.min_depth);
    }

    /// Removes the rightmost leaf from the merkle
    /// Currently O(n) in the number of leaves (could be log(n))
    pub fn pop_leaf(&mut self) {
        let mut leaves = self.layers.lock().data.swap_remove(0);
        leaves.pop();
        *self = Self::new_advanced(self.ty, leaves, self.min_depth);
    }

    // Sets the leaf at the given index to the given hash.
    // Panics if the index is out of bounds (since the structure doesn't grow).
    pub fn set(&self, idx: usize, hash: Bytes32) {
        #[cfg(feature = "counters")]
        SET_COUNTERS[&self.ty].fetch_add(1, Ordering::Relaxed);
        let mut layers = self.layers.lock();
        if layers.data[0][idx] == hash {
            return;
        }
        layers.data[0][idx] = hash;
        layers.dirty_leaf_parents.set(idx >> 1, true);
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
        let mut layers = self.layers.lock();
        let start = layers.data[0].len();
        let mut layer_size = new_len;
        for (layer_i, layer) in layers.data.iter_mut().enumerate() {
            layer.resize(layer_size, *empty_hash_at(self.ty, layer_i));
            layer_size = max(layer_size >> 1, 1);
        }
        layers.dirty_leaf_parents[(start >> 1)..].fill(true);
        layers.dirty_leaf_parents.resize(new_len >> 1, true);
        Ok(layers.data[0].len())
    }
}

impl PartialEq for Merkle {
    // There are only three members of a Merkle, the type, the layers, and the min_depth.
    //
    // It should be obvious that only if the type and layers are equal, will the root hash
    // be equal. So, it is sufficient to compare the root hash when checking equality.
    //
    // However, it is possible that the min_depth may differ between two merkle trees which
    // have the same type and layers. The root hash will still be equal unless the min_depth
    // is larger than the depth required to hold the data in the layers.
    //
    // For example, a Merkle tree with 5 leaves requires 3 layeers to hold the data. If the
    // min_depth is 1 on one tree and 2 on another, the root has would still be equal
    // because the same nodes are hashed together. However, the min_dpeth was 4, then,
    // there would be 4 layers in that tree, and the root hash would be different.
    fn eq(&self, other: &Self) -> bool {
        self.root() == other.root()
    }
}

impl Eq for Merkle {}

pub mod mutex_sedre {
    use parking_lot::Mutex;

    pub fn serialize<S, T>(data: &Mutex<T>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: serde::Serializer,
        T: serde::Serialize,
    {
        data.lock().serialize(serializer)
    }

    pub fn deserialize<'de, D, T>(deserializer: D) -> Result<Mutex<T>, D::Error>
    where
        D: serde::Deserializer<'de>,
        T: serde::Deserialize<'de>,
    {
        Ok(Mutex::new(T::deserialize(deserializer)?))
    }
}

#[cfg(test)]
mod test {
    use super::*;
    use crate::memory;
    use arbutil::Bytes32;
    use core::panic;
    use enum_iterator::all;

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
        assert_eq!(merkle.root(), expected);

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
        let merkle: Merkle = Merkle::new(MerkleType::Value, vec![]);
        assert_eq!(merkle.capacity(), 0);
        let merkle = Merkle::new(MerkleType::Value, vec![Bytes32::from([1; 32])]);
        assert_eq!(merkle.capacity(), 1);
        let merkle = Merkle::new(
            MerkleType::Value,
            vec![Bytes32::from([1; 32]), Bytes32::from([2; 32])],
        );
        assert_eq!(merkle.capacity(), 2);
        let merkle = Merkle::new_advanced(MerkleType::Memory, vec![Bytes32::from([1; 32])], 11);
        assert_eq!(merkle.capacity(), 1024);
    }

    #[test]
    #[ignore = "This is just used for generating the zero hashes for the memory merkle trees."]
    fn emit_memory_zerohashes() {
        // The following code was generated from the empty_leaf_hash() test in the memory package.
        let mut empty_node = Bytes32([
            57, 29, 211, 154, 252, 227, 18, 99, 65, 126, 203, 166, 252, 232, 32, 3, 98, 194, 254,
            186, 118, 14, 139, 192, 101, 156, 55, 194, 101, 11, 11, 168,
        ])
        .clone();
        for _ in 0..64 {
            print!("Bytes32([");
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
    fn clone_is_separate() {
        let merkle = Merkle::new_advanced(MerkleType::Value, vec![Bytes32::from([1; 32])], 4);
        let m2 = merkle.clone();
        m2.resize(4).expect("resize failed");
        m2.set(3, Bytes32::from([2; 32]));
        assert_ne!(merkle, m2);
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

    #[test]
    fn test_zero_hashes() {
        for ty in all::<MerkleType>() {
            if ty == MerkleType::Empty {
                continue;
            }
            let mut empty_hash = Bytes32::from([0; 32]);
            if ty == MerkleType::Memory {
                empty_hash = memory::testing::empty_leaf_hash();
            }
            for layer in 0..64 {
                // empty_hash_at is just a lookup, but empty_hash is calculated iteratively.
                assert_eq!(empty_hash_at(ty, layer), &empty_hash);
                empty_hash = hash_node(ty, &empty_hash, &empty_hash);
            }
        }
    }
}
