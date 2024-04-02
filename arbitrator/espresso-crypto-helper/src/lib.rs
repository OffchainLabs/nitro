mod bytes;
mod namespace;

use ark_bn254::Bn254;
use ark_serialize::CanonicalDeserialize;
use jf_primitives::{
    merkle_tree::{
        prelude::LightWeightSHA3MerkleTree, AppendableMerkleTreeScheme, MerkleTreeScheme,
    },
    pcs::prelude::UnivariateUniversalParams,
    vid::{advz::Advz, VidScheme as VidSchemeTrait},
};
use lazy_static::lazy_static;
use namespace::{NameSpaceTable, NamespaceProof, Transaction, TxTableEntryWord};
use sha2::{Digest, Sha256};
use tagged_base64::TaggedBase64;

use crate::bytes::Bytes;

pub type VidScheme = Advz<Bn254, sha2::Sha256>;
pub type BlockMerkleTree = LightWeightSHA3MerkleTree<u64>;

lazy_static! {
    // Initialize the byte array from JSON content
    static ref SRS_VEC: Vec<u8> = {
        let json_content = include_str!("../../../config/vid_srs.json");
        serde_json::from_str(json_content).expect("Failed to deserialize")
    };
}

pub fn verify_merkle_proof_helper(
    _root_bytes: &[u8],
    _proof_bytes: &[u8],
    _block_comm_bytes: &[u8],
) {
    let mut tree = BlockMerkleTree::from_elems(Some(5), vec![1]).expect("should construct tree");
    tree.push(1).unwrap();
    tree.lookup(1).expect_ok().unwrap();
}

// Helper function to verify a VID namespace proof that takes the byte representations of the proof,
// namespace table, and commitment string.
//
// proof_bytes: Byte representation of a JSON NamespaceProof string
// commit_bytes: Byte representation of a TaggedBase64 payload commitment string
// ns_table_bytes: Raw bytes of the namespace table
// tx_comm_bytes: Byte representation of a hex encoded Sha256 digest that the transaction set commits to
pub fn verify_namespace_helper(
    namespace: u64,
    proof_bytes: &[u8],
    commit_bytes: &[u8],
    ns_table_bytes: &[u8],
    tx_comm_bytes: &[u8],
) {
    let proof_str = std::str::from_utf8(proof_bytes).unwrap();
    let commit_str = std::str::from_utf8(commit_bytes).unwrap();
    let txn_comm_str = std::str::from_utf8(tx_comm_bytes).unwrap();

    let proof: NamespaceProof = serde_json::from_str(proof_str).unwrap();
    let ns_table = NameSpaceTable::<TxTableEntryWord>::from_bytes(<&[u8] as Into<Bytes>>::into(
        ns_table_bytes,
    ));
    let tagged = TaggedBase64::parse(&commit_str).unwrap();
    let commit: <VidScheme as VidSchemeTrait>::Commit = tagged.try_into().unwrap();

    let srs =
        UnivariateUniversalParams::<Bn254>::deserialize_uncompressed_unchecked(SRS_VEC.as_slice())
            .unwrap();
    let num_storage_nodes = match &proof {
        NamespaceProof::Existence { vid_common, .. } => {
            VidScheme::get_num_storage_nodes(&vid_common)
        }
        // Non-existence proofs do not actually make use of the SRS, so pick some random value to appease the compiler.
        _ => 5,
    };
    let num_chunks: usize = 1 << num_storage_nodes.ilog2();
    let advz = Advz::new(num_chunks, num_storage_nodes, 1, srs).unwrap();
    let (txns, ns) = proof.verify(&advz, &commit, &ns_table).unwrap();

    let txns_comm = hash_txns(namespace, &txns);

    assert!(ns == namespace.into());
    assert!(txns_comm == txn_comm_str);
}

// TODO: Use Commit trait: https://github.com/EspressoSystems/nitro-espresso-integration/issues/88
fn hash_txns(namespace: u64, txns: &[Transaction]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(namespace.to_le_bytes());
    for txn in txns {
        hasher.update(&txn.payload);
    }
    let hash_result = hasher.finalize();
    format!("{:x}", hash_result)
}

#[cfg(test)]
mod test {
    use super::*;
    use ark_serialize::CanonicalSerialize;
    use jf_primitives::pcs::{
        checked_fft_size, prelude::UnivariateKzgPCS, PolynomialCommitmentScheme,
    };
    #[test]
    fn test_verify_namespace_helper() {
        let proof_bytes = b"{\"NonExistence\":{\"ns_id\":0}}";
        let commit_bytes = b"HASH~1yS-KEtL3oDZDBJdsW51Pd7zywIiHesBZsTbpOzrxOfu";
        let txn_comm_str = hash_txns(0, &[]);
        let txn_comm_bytes = txn_comm_str.as_bytes();
        let ns_table_bytes = &[0, 0, 0, 0];
        verify_namespace_helper(0, proof_bytes, commit_bytes, ns_table_bytes, txn_comm_bytes);
    }

    #[test]
    fn test_verify_merkle_proof_helper() {
        verify_merkle_proof_helper(&[], &[], &[])
    }

    #[test]
    fn write_srs_to_file() {
        let mut bytes = Vec::new();
        let mut rng = jf_utils::test_rng();
        UnivariateKzgPCS::<Bn254>::gen_srs_for_testing(&mut rng, checked_fft_size(200).unwrap())
            .unwrap()
            .serialize_uncompressed(&mut bytes)
            .unwrap();
        let _ = std::fs::write("srs.json", serde_json::to_string(&bytes).unwrap());
    }
}
