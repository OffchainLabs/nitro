// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use digest::Digest;
// use rayon::prelude::*;
use sha3::Keccak256;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
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

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Merkle {
    tree: Vec<u8>,
    empty_hash: Bytes32,
}

#[inline]
fn hash_node(a: &[u8], b: &[u8]) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(a);
    h.update(b);
    h.finalize().into()
}

impl Merkle {
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

        let empty_layer_hash = hash_node(empty_hash.as_slice(), empty_hash.as_slice());

        let hash_count = hashes.len();
        let mut current_level_size = hash_count;

        // Calculate the total capacity needed for the tree
        let mut total_capacity = hash_count * 32; // 32 bytes per hash
        let mut depth = min_depth;
        while current_level_size > 1 || depth > 0 {
            current_level_size = (current_level_size + 1) / 2;
            total_capacity += current_level_size * 32;
            depth = depth.saturating_sub(1);
        }
        let mut tree = Vec::with_capacity(total_capacity);

        // Append initial hashes to the tree
        for hash in hashes.into_iter() {
            tree.extend_from_slice(hash.as_slice());
        }

        let mut next_level_offset = tree.len();
        let mut depth = min_depth;

        while current_level_size > 1 || depth > 0 {
            let mut i = next_level_offset - current_level_size * 32;
            while i < next_level_offset {
                let left = &tree[i..i + 32];
                let right = if i + 32 < next_level_offset {
                    &tree[i + 32..i + 64]
                } else {
                    empty_layer_hash.as_slice()
                };

                let parent_hash = hash_node(left, right);
                tree.extend(parent_hash.as_slice());

                i += 64;
            }

            current_level_size = (current_level_size + 1) / 2;
            next_level_offset = tree.len();
            depth = depth.saturating_sub(1);
        }

        Merkle {
            tree,
            empty_hash: empty_layer_hash,
        }
    }

    pub fn root(&self) -> Bytes32 {
        let len = self.tree.len();
        let mut root = [0u8; 32];
        root.copy_from_slice(&self.tree[len - 32..len]);
        root.into()
    }

    pub fn leaves(&self) -> &[u8] {
        let leaf_layer_size = self.calculate_layer_size(0);
        &self.tree[..leaf_layer_size * 32]
    }

    pub fn prove(&self, idx: usize) -> Option<Vec<u8>> {
        let leaf_count = self.calculate_layer_size(0);
        if idx >= leaf_count {
            return None;
        }

        let mut proof = Vec::new();
        let mut node_index = idx;
        let mut layer_start = 0;

        for depth in 0.. {
            let layer_size = self.calculate_layer_size(depth);
            if layer_size <= 1 {
                break;
            }

            let sibling_index = if node_index % 2 == 0 {
                node_index + 1
            } else {
                node_index - 1
            };
            if sibling_index < layer_size {
                proof.extend(self.get_node(layer_start, sibling_index));
            }

            node_index /= 2;
            layer_start += layer_size * 32;
        }

        Some(proof)
    }

    // Helper function to get a node from the tree
    #[inline(always)]
    fn get_node(&self, layer_start: usize, index: usize) -> Bytes32 {
        let start = layer_start + index * 32;
        let mut node = [0u8; 32];
        node.copy_from_slice(&self.tree[start..start + 32]);
        node.into()
    }

    pub fn set(&mut self, mut idx: usize, hash: Bytes32) {
        // Calculate the offset in the flat tree for the given index
        let mut offset = idx * 32;

        // Check if the hash at the calculated position is the same as the input hash
        if &self.tree[offset..offset + 32] == hash.as_slice() {
            return;
        }

        // Copy the new hash into the tree at the calculated position
        self.tree[offset..offset + 32].copy_from_slice(hash.as_slice());

        // Calculate the total number of nodes in the tree
        let total_nodes = self.tree.len() / 32;

        // Update parent hashes up the tree
        let mut next_hash = hash;
        while idx > 0 {
            idx = (idx - 1) / 2; // Move to the parent index
            offset = idx * 32;

            // Calculate the position of the sibling in the flat tree
            let sibling_idx = if idx % 2 == 0 { idx + 1 } else { idx - 1 };
            let sibling_offset = sibling_idx * 32;

            // Handle the case where the sibling index is out of bounds
            let sibling_hash = if sibling_offset < total_nodes * 32 {
                &self.tree[sibling_offset..sibling_offset + 32]
            } else {
                self.empty_hash.as_slice()
            };

            // Calculate the parent hash
            next_hash = if idx % 2 == 0 {
                hash_node(next_hash.as_slice(), sibling_hash)
            } else {
                hash_node(sibling_hash, next_hash.as_slice())
            };

            // Update the parent node in the flat tree
            self.tree[offset..offset + 32].copy_from_slice(next_hash.as_slice());
        }
    }

    // Helper function to calculate the size of a given layer
    #[inline(always)]
    fn calculate_layer_size(&self, depth: usize) -> usize {
        let mut size = self.tree.len() / 32; // Total number of nodes
        for _ in 0..depth {
            size = (size + 1) / 2; // Size of the current layer
        }
        size
    }
}
