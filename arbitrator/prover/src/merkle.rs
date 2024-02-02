// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::utils::Bytes32;
use digest::Digest;
//use rayon::prelude::*;
use sha3::Keccak256;
use std::convert::TryFrom;

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
    ty: MerkleType,
    layers: Vec<Vec<Bytes32>>,
    empty_layers: Vec<Bytes32>,
}

fn hash_node(ty: MerkleType, a: Bytes32, b: Bytes32) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(ty.get_prefix());
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
        let mut layers = vec![hashes];
        let mut empty_layers = vec![empty_hash];
        while layers.last().unwrap().len() > 1 || layers.len() < min_depth {
            let empty_layer = *empty_layers.last().unwrap();
            let new_layer = layers
                .last()
                .unwrap()
                .chunks(2)
                .map(|window| {
                    hash_node(ty, window[0], window.get(1).cloned().unwrap_or(empty_layer))
                })
                .collect();
            empty_layers.push(hash_node(ty, empty_layer, empty_layer));
            layers.push(new_layer);
        }
        Merkle {
            ty,
            layers,
            empty_layers,
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
    pub fn prove(&self, mut idx: usize) -> Option<Vec<u8>> {
        if idx >= self.leaves().len() {
            return None;
        }
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
        Some(proof)
    }

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
