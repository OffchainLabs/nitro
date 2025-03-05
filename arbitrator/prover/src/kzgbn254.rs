use crate::utils::append_left_padded_uint32_be;
use crate::{utils::append_left_padded_biguint_be, Bytes32};
use ark_bn254::G2Affine;
use ark_ec::{AffineRepr, CurveGroup};
use ark_ff::{BigInteger, PrimeField};
use eyre::{ensure, Result};
use kzgbn254::{blob::Blob, kzg::Kzg, polynomial::PolynomialFormat};
use num::BigUint;
use sha2::Digest;
use sha3::Keccak256;
use std::env;
use std::io::Write;
use std::path::PathBuf;

lazy_static::lazy_static! {
    // srs_points_to_load = 131072 (65536 is enough)

    pub static ref KZG_BN254_SETTINGS: Kzg = Kzg::setup(
        &load_directory_with_prefix("src/mainnet-files/g1.point.65536"),
        &load_directory_with_prefix("src/mainnet-files/g2.point.65536"),
        &load_directory_with_prefix("src/mainnet-files/g2.point.powerOf2"),
        268435456,
        65536
    ).unwrap();
}

// Necessary helper function for understanding if srs is being loaded for normal node operation
// or for challenge testing.
fn load_directory_with_prefix(directory_name: &str) -> String {
    let cwd = env::current_dir().expect("Failed to get current directory");

    let path = if cwd.ends_with("system_tests") {
        PathBuf::from("../arbitrator/prover/").join(directory_name)
    } else {
        PathBuf::from("./arbitrator/prover/").join(directory_name)
    };

    path.to_string_lossy().into_owned()
}

/// Creates a KZG preimage proof consumable by the point evaluation precompile.
pub fn prove_kzg_preimage_bn254(
    hash: Bytes32,
    preimage: &[u8],
    offset: u32,
    out: &mut impl Write,
) -> Result<()> {
    let mut kzg = KZG_BN254_SETTINGS.clone();
    // expand roots of unity
    kzg.calculate_roots_of_unity(preimage.len() as u64)?;

    // preimage is already padded and is the actual blob data, NOT the IFFT'd form.
    let blob = Blob::from_padded_bytes_unchecked(&preimage);

    let blob_polynomial_evaluation_form =
        blob.to_polynomial(PolynomialFormat::InCoefficientForm)?;
    let blob_commitment = kzg.commit(&blob_polynomial_evaluation_form)?;

    let commitment_x_bigint: BigUint = blob_commitment.x.into();
    let commitment_y_bigint: BigUint = blob_commitment.y.into();
    let length_uint32_fe: u32 = (blob.len() as u32) / 32;

    let mut commitment_encoded_length_bytes = Vec::with_capacity(68);
    append_left_padded_biguint_be(&mut commitment_encoded_length_bytes, &commitment_x_bigint);
    append_left_padded_biguint_be(&mut commitment_encoded_length_bytes, &commitment_y_bigint);
    append_left_padded_uint32_be(&mut commitment_encoded_length_bytes, &length_uint32_fe);

    let mut keccak256_hasher = Keccak256::new();
    keccak256_hasher.update(&commitment_encoded_length_bytes);
    let commitment_hash: Bytes32 = keccak256_hasher.finalize().into();

    ensure!(
        hash == commitment_hash,
        "Trying to prove versioned hash {} preimage but recomputed hash {}",
        hash,
        commitment_hash,
    );

    ensure!(
        offset % 32 == 0,
        "Cannot prove blob preimage at unaligned offset {}",
        offset,
    );

    let mut commitment_encoded_bytes = Vec::with_capacity(64);

    append_left_padded_biguint_be(&mut commitment_encoded_bytes, &commitment_x_bigint);
    append_left_padded_biguint_be(&mut commitment_encoded_bytes, &commitment_y_bigint);

    let mut proving_offset = offset;
    let length_usize_32 = (preimage.len() / 32) as u32;

    assert!(length_usize_32 == blob_polynomial_evaluation_form.len() as u32);

    // address proving past end edge case later
    // offset refers to a 32 byte section or field element of the blob
    let proving_past_end = offset >= preimage.len() as u32;
    if proving_past_end {
        // Proving any offset proves the length which is all we need here,
        // because we're past the end of the preimage.
        proving_offset = 0;
    }

    // Y = ϕ(offset)
    let proven_y_fr = blob_polynomial_evaluation_form
        .get_at_index(proving_offset as usize / 32)
        .ok_or_else(|| {
            eyre::eyre!(
                "Index ({}) out of bounds for preimage of length {} with data of ({} field elements x 32 bytes)",
                proving_offset,
                preimage.len(),
                blob_polynomial_evaluation_form.len()
            )
        })?;

    let z_fr = kzg
        .get_nth_root_of_unity(proving_offset as usize / 32)
        .ok_or_else(|| eyre::eyre!("Failed to get nth root of unity"))?;

    let proven_y = proven_y_fr.into_bigint().to_bytes_be();
    let z = z_fr.into_bigint().to_bytes_be();

    // probably should be a constant on the contract.
    let g2_generator = G2Affine::generator();
    let z_g2 = (g2_generator * z_fr).into_affine();

    // if we are loading in g2 pow2 this is index 0 not 1
    let g2_tau: G2Affine = kzg
        .get_g2_points()
        .get(1)
        .ok_or_else(|| eyre::eyre!("Failed to get g2 point at index 1 in SRS"))?
        .clone();
    let g2_tau_minus_g2_z = (g2_tau - z_g2).into_affine();

    let kzg_proof = kzg.compute_kzg_proof_with_roots_of_unity(
        &blob_polynomial_evaluation_form,
        proving_offset as u64 / 32,
    )?;

    let offset_usize = proving_offset as usize;
    // This should cause failure when proving past offset.
    if !proving_past_end {
        ensure!(
            *proven_y == preimage[offset_usize..offset_usize + 32],
            "KZG proof produced wrong preimage for offset {}",
            offset,
        );
    }

    /*
        Encode the machine state proof used for resolving a
        one step proof for EigenDA preimage types.
     */

    let xminusz_x0: BigUint = g2_tau_minus_g2_z.x.c0.into();
    let xminusz_x1: BigUint = g2_tau_minus_g2_z.x.c1.into();
    let xminusz_y0: BigUint = g2_tau_minus_g2_z.y.c0.into();
    let xminusz_y1: BigUint = g2_tau_minus_g2_z.y.c1.into();

    // turn each element of xminusz into bytes, then pad each to 32 bytes, then append in order x1,x0,y1,y0
    let mut xminusz_encoded_bytes = Vec::with_capacity(128);
    append_left_padded_biguint_be(&mut xminusz_encoded_bytes, &xminusz_x1);
    append_left_padded_biguint_be(&mut xminusz_encoded_bytes, &xminusz_x0);
    append_left_padded_biguint_be(&mut xminusz_encoded_bytes, &xminusz_y1);
    append_left_padded_biguint_be(&mut xminusz_encoded_bytes, &xminusz_y0);

    // encode the kzg point opening proof
    let proof_x_bigint: BigUint = kzg_proof.x.into();
    let proof_y_bigint: BigUint = kzg_proof.y.into();
    let mut proof_encoded_bytes = Vec::with_capacity(64);
    append_left_padded_biguint_be(&mut proof_encoded_bytes, &proof_x_bigint);
    append_left_padded_biguint_be(&mut proof_encoded_bytes, &proof_y_bigint);

    // encode the number of field elements in the blob
    let mut length_fe_bytes = Vec::with_capacity(32);
    append_left_padded_biguint_be(&mut length_fe_bytes, &BigUint::from(length_usize_32));

    out.write_all(&*z)?; // evaluation point [:32]
    out.write_all(&*proven_y)?; // expected output [32:64]
    out.write_all(&xminusz_encoded_bytes)?; // g2TauMinusG2z [64:192]
    out.write_all(&*commitment_encoded_bytes)?; // kzg commitment [192:256]
    out.write_all(&proof_encoded_bytes)?; // proof [256:320]
    out.write_all(&*length_fe_bytes)?; // length of preimage [320:352]

    Ok(())
}
