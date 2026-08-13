use alloy_primitives::B256;

use crate::{Blob, Result};

const VERSIONED_HASH_VERSION_KZG: u8 = 0x01;

/// Computes a blob's KZG commitment and derives its EIP-4844 versioned hash.
pub fn blob_to_versioned_hash(blob: &Blob) -> Result<B256> {
    let kzg_blob = c_kzg::Blob::new(blob.0);
    let commitment = c_kzg::ethereum_kzg_settings(0).blob_to_kzg_commitment(&kzg_blob)?;
    Ok(kzg_commitment_to_versioned_hash(
        commitment.to_bytes().as_slice(),
    ))
}

/// `sha256(commitment)` with byte 0 set to the KZG version tag.
/// Mirrors Nitro's `blobs.CommitmentToVersionedHash`.
pub fn kzg_commitment_to_versioned_hash(commitment: &[u8]) -> B256 {
    use sha2::{Digest, Sha256};

    let mut hash = Sha256::digest(commitment);
    hash[0] = VERSIONED_HASH_VERSION_KZG;
    B256::from_slice(&hash)
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Compressed G1 identity point: the KZG commitment of an all-zero blob.
    fn zero_blob_commitment() -> [u8; 48] {
        let mut c = [0u8; 48];
        c[0] = 0xc0;
        c
    }

    #[test]
    fn versioned_hash_sets_kzg_version_byte() {
        let vh = kzg_commitment_to_versioned_hash(&[0u8; 48]);
        assert_eq!(vh[0], VERSIONED_HASH_VERSION_KZG);
    }

    #[test]
    fn zero_blob_matches_known_commitment_hash() {
        let got = blob_to_versioned_hash(&Blob::default()).unwrap();
        let expected = kzg_commitment_to_versioned_hash(&zero_blob_commitment());

        assert_eq!(got, expected);
        assert_eq!(got[0], VERSIONED_HASH_VERSION_KZG);
    }
}
