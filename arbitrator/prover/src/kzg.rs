// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::utils::Bytes32;
use c_kzg::{KzgSettings, BYTES_PER_G1_POINT, BYTES_PER_G2_POINT};
use eyre::{ensure, Result, WrapErr};
use num::BigUint;
use serde::{de::Error as _, Deserialize};
use sha2::{Digest, Sha256};
use std::{convert::TryFrom, io::Write};

struct HexBytesParser;

impl<'de, const N: usize> serde_with::DeserializeAs<'de, [u8; N]> for HexBytesParser {
    fn deserialize_as<D>(deserializer: D) -> Result<[u8; N], D::Error>
    where
        D: serde::Deserializer<'de>,
    {
        let s = String::deserialize(deserializer)?;
        let mut s = s.as_str();
        if s.starts_with("0x") {
            s = &s[2..];
        }
        let mut bytes = [0; N];
        match hex::decode_to_slice(s, &mut bytes) {
            Ok(()) => Ok(bytes),
            Err(err) => Err(D::Error::custom(err.to_string())),
        }
    }
}

#[derive(Deserialize)]
struct TrustedSetup {
    #[serde(with = "serde_with::As::<Vec<HexBytesParser>>")]
    g1_lagrange: Vec<[u8; BYTES_PER_G1_POINT]>,
    #[serde(with = "serde_with::As::<Vec<HexBytesParser>>")]
    g2_monomial: Vec<[u8; BYTES_PER_G2_POINT]>,
}

const FIELD_ELEMENTS_PER_BLOB: usize = 4096;

lazy_static::lazy_static! {
    pub static ref ETHEREUM_KZG_SETTINGS: KzgSettings = {
        let trusted_setup = serde_json::from_str::<TrustedSetup>(include_str!("kzg-trusted-setup.json"))
            .expect("Failed to deserialize Ethereum trusted setup");
        KzgSettings::load_trusted_setup(&trusted_setup.g1_lagrange, &trusted_setup.g2_monomial)
            .expect("Failed to load Ethereum trusted setup")
    };

    pub static ref BLS_MODULUS: BigUint = "52435875175126190479447740508185965837690552500527637822603658699938581184513".parse().unwrap();
    pub static ref ROOT_OF_UNITY: BigUint = {
        // order 2^32
        let root: BigUint = "10238227357739495823651030575849232062558860180284477541189508159991286009131".parse().unwrap();
        let exponent = (1_u64 << 32) / (FIELD_ELEMENTS_PER_BLOB as u64);
        root.modpow(&BigUint::from(exponent), &BLS_MODULUS)
    };
}

/// Creates a KZG preimage proof consumable by the point evaluation precompile.
pub fn prove_kzg_preimage(
    hash: Bytes32,
    preimage: &[u8],
    offset: u32,
    out: &mut impl Write,
) -> Result<()> {
    let blob =
        c_kzg::Blob::from_bytes(preimage).wrap_err("Failed to generate KZG blob from preimage")?;
    let commitment = c_kzg::KzgCommitment::blob_to_kzg_commitment(&blob, &ETHEREUM_KZG_SETTINGS)
        .wrap_err("Failed to generate KZG commitment from blob")?;
    let mut expected_hash: Bytes32 = Sha256::digest(&*commitment).into();
    expected_hash[0] = 1;
    ensure!(
        hash == expected_hash,
        "Trying to prove versioned hash {} preimage but recomputed hash {}",
        hash,
        expected_hash,
    );
    ensure!(
        offset % 32 == 0,
        "Cannot prove blob preimage at unaligned offset {}",
        offset,
    );
    let offset_usize = usize::try_from(offset)?;
    let mut proving_offset = offset;
    let proving_past_end = offset_usize >= preimage.len();
    if proving_past_end {
        // Proving any offset proves the length which is all we need here,
        // because we're past the end of the preimage.
        proving_offset = 0;
    }
    let exp = (proving_offset / 32).reverse_bits()
        >> (u32::BITS - FIELD_ELEMENTS_PER_BLOB.trailing_zeros());
    let z = ROOT_OF_UNITY.modpow(&BigUint::from(exp), &BLS_MODULUS);
    let z_bytes = z.to_bytes_be();
    let mut padded_z_bytes = [0u8; 32];
    padded_z_bytes[32 - z_bytes.len()..].copy_from_slice(&z_bytes);
    let z_bytes = c_kzg::Bytes32::from(padded_z_bytes);
    let (kzg_proof, proven_y) =
        c_kzg::KzgProof::compute_kzg_proof(&blob, &z_bytes, &ETHEREUM_KZG_SETTINGS)
            .wrap_err("Failed to generate KZG proof from blob and z")?;
    if !proving_past_end {
        ensure!(
            *proven_y == preimage[offset_usize..offset_usize + 32],
            "KZG proof produced wrong preimage for offset {}",
            offset,
        );
    }
    out.write_all(&*hash)?;
    out.write_all(&*z_bytes)?;
    out.write_all(&*proven_y)?;
    out.write_all(&*commitment)?;
    out.write_all(kzg_proof.to_bytes().as_slice())?;
    Ok(())
}

#[cfg(test)]
#[test]
fn load_trusted_setup() {
    let _: &KzgSettings = &*ETHEREUM_KZG_SETTINGS;
}
