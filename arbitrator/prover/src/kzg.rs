// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::Bytes32;
use c_kzg::{KzgSettings, BYTES_PER_BLOB, FIELD_ELEMENTS_PER_BLOB};
use eyre::{ensure, Result, WrapErr};
use num::BigUint;
use sha2::{Digest, Sha256};
use std::{convert::TryFrom, io::Write};

lazy_static::lazy_static! {
    pub static ref ETHEREUM_KZG_SETTINGS: KzgSettings = {
        let trusted_setup = include_str!("kzg-trusted-setup.txt");
        KzgSettings::parse_kzg_trusted_setup(trusted_setup, 0)
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
    ensure!(
        preimage.len() == BYTES_PER_BLOB,
        "Trying to KZG prove preimage of unexpected length {}",
        preimage.len(),
    );
    let blob =
        c_kzg::Blob::from_bytes(preimage).wrap_err("Failed to generate KZG blob from preimage")?;
    let commitment = ETHEREUM_KZG_SETTINGS
        .blob_to_kzg_commitment(&blob)
        .wrap_err("Failed to generate KZG commitment from blob")?;
    let mut expected_hash: Bytes32 = Sha256::digest(*commitment).into();
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
    let (kzg_proof, proven_y) = ETHEREUM_KZG_SETTINGS
        .compute_kzg_proof(&blob, &z_bytes)
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
    let _: &KzgSettings = &ETHEREUM_KZG_SETTINGS;
}
