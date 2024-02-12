// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::utils::Bytes32;
// use digest::Digest;
use rayon::prelude::*;
use sha3::{Digest, Keccak256};
use std::convert::TryFrom;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Merkle {
    layers: Vec<Vec<Bytes32>>,
    empty_layers: Vec<Bytes32>,
}

fn hash_node(a: Bytes32, b: Bytes32) -> Bytes32 {
    let mut h = Keccak256::new();
    h.update(a);
    h.update(b);
    h.finalize().into()
}

impl Merkle {
    pub fn new_advanced(hashes: Vec<Bytes32>, empty_hash: Bytes32, min_depth: usize) -> Merkle {
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
                .map(|window| hash_node(window[0], window.get(1).cloned().unwrap_or(empty_layer)))
                .collect();
            empty_layers.push(hash_node(empty_layer, empty_layer));
            layers.push(new_layer);
        }
        Merkle {
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
}
