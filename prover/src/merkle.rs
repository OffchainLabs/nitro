use crate::utils::Bytes32;
use digest::Digest;
use sha3::Keccak256;
use std::convert::TryInto;

#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Merkle {
    layers: Vec<Vec<Bytes32>>,
}

impl Merkle {
    pub fn new(hashes: Vec<Bytes32>) -> Merkle {
        if hashes.is_empty() {
            return Merkle::default();
        }
        let mut layers = Vec::new();
        layers.push(hashes);
        while layers.last().unwrap().len() > 1 {
            let mut new_layer = Vec::new();
            for window in layers.last().unwrap().chunks(2) {
                let mut h = Keccak256::new();
                h.update("Value merkle tree:");
                h.update(window[0]);
                h.update(window.get(1).cloned().unwrap_or_default());
                new_layer.push(h.finalize().into());
            }
            layers.push(new_layer);
        }
        Merkle { layers }
    }

    pub fn root(&self) -> Bytes32 {
        if let Some(layer) = self.layers.last() {
            assert_eq!(layer.len(), 1);
            layer[0]
        } else {
            Bytes32::default()
        }
    }

	#[must_use]
    pub fn prove(&self, mut idx: usize) -> Vec<u8> {
        let mut proof = Vec::new();
        proof.push(self.layers.len().try_into().unwrap());
        for layer in &self.layers {
			if layer.len() <= 1 {
				break;
			}
            let counterpart = idx ^ 1;
            proof.extend(layer.get(counterpart).cloned().unwrap_or_default());
            idx >>= 1;
        }
        proof
    }
}
