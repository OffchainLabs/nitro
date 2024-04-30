mod bytes;
mod sequencer_data_structures;

use ark_bn254::Bn254;
use ark_ff::PrimeField;
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use committable::{Commitment, Committable};
use ethers_core::types::U256;
use jf_primitives::{
    crhf::{VariableLengthRescueCRHF, CRHF},
    errors::PrimitivesError,
    merkle_tree::{
        prelude::{MerkleNode, MerkleProof, Sha3Node},
        MerkleCommitment, MerkleTreeScheme,
    },
    pcs::prelude::UnivariateUniversalParams,
    vid::{advz::Advz, VidScheme as VidSchemeTrait},
};
use lazy_static::lazy_static;
use sequencer_data_structures::{
    BlockMerkleTree, Header, NameSpaceTable, NamespaceProof, Transaction, TxTableEntryWord,
};
use sha2::{Digest, Sha256};
use tagged_base64::TaggedBase64;

use crate::{
    bytes::Bytes,
    sequencer_data_structures::{field_to_u256, BlockMerkleCommitment},
};

pub type VidScheme = Advz<Bn254, sha2::Sha256>;
pub type Proof = Vec<MerkleNode<Commitment<Header>, u64, Sha3Node>>;
pub type CircuitField = ark_ed_on_bn254::Fq;

lazy_static! {
    // Initialize the byte array from JSON content
    static ref SRS_VEC: Vec<u8> = {
        let json_content = include_str!("../../../config/vid_srs.json");
        serde_json::from_str(json_content).expect("Failed to deserialize")
    };
}

// Helper function to verify a block merkle proof.
//
// proof_bytes: Byte representation of a block merkle proof.
// root_bytes: Byte representation of a Sha3Node merkle root.
// header_bytes: Byte representation of the HotShot header being validated as a Merkle leaf.
// circuit_block_bytes: Circuit representation of the HotShot header commitment returned by the light client contract.
pub fn verify_merkle_proof_helper(
    proof_bytes: &[u8],
    header_bytes: &[u8],
    block_comm_bytes: &[u8],
    circuit_block_bytes: &[u8],
) {
    let proof_str = std::str::from_utf8(proof_bytes).unwrap();
    let header_str = std::str::from_utf8(header_bytes).unwrap();
    let block_comm_str = std::str::from_utf8(block_comm_bytes).unwrap();
    let tagged = TaggedBase64::parse(&block_comm_str).unwrap();
    let block_comm: BlockMerkleCommitment = tagged.try_into().unwrap();

    let proof: Proof = serde_json::from_str(proof_str).unwrap();
    let header: Header = serde_json::from_str(header_str).unwrap();
    let header_comm: Commitment<Header> = header.commit();

    let proof = MerkleProof::new(header.height, proof.to_vec());
    let proved_comm = proof.elem().unwrap().clone();
    BlockMerkleTree::verify(block_comm.digest(), header.height, proof)
        .unwrap()
        .unwrap();

    let mut block_comm_root_bytes = vec![];
    block_comm
        .serialize_compressed(&mut block_comm_root_bytes)
        .unwrap();
    let field_bytes = hash_bytes_to_field(&block_comm_root_bytes).unwrap();
    let local_block_comm_u256 = field_to_u256(field_bytes);
    let circuit_block_comm_u256 = U256::from_little_endian(circuit_block_bytes);

    assert!(proved_comm == header_comm);
    assert!(local_block_comm_u256 == circuit_block_comm_u256)
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

fn hash_bytes_to_field(bytes: &[u8]) -> Result<CircuitField, PrimitivesError> {
    // make sure that `mod_order` won't happen.
    let bytes_len = ((<CircuitField as PrimeField>::MODULUS_BIT_SIZE + 7) / 8 - 1) as usize;
    let elem = bytes
        .chunks(bytes_len)
        .map(CircuitField::from_le_bytes_mod_order)
        .collect::<Vec<_>>();
    Ok(VariableLengthRescueCRHF::<_, 1>::evaluate(elem)?[0])
}

#[cfg(test)]
mod test {
    use super::*;
    use jf_primitives::pcs::{
        checked_fft_size, prelude::UnivariateKzgPCS, PolynomialCommitmentScheme,
    };

    lazy_static! {
        // Initialize the byte array from JSON content
        static ref PROOF: &'static str = {
            include_str!("../../../config/test_merkle_path.json")
        };
        static ref HEADER: &'static str = {
            include_str!("../../../config/test_header.json")
        };
    }
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
        let proof_bytes = PROOF.clone().as_bytes();
        let header_bytes = HEADER.clone().as_bytes();
        let block_comm_bytes =
            b"MERKLE_COMM~vc7j-uHdU6RGWMlKRVReWs5VGn_vuG-F-0s-jZ2eUa0gAAAAAAAAAAIAAAAAAAAAvQ";
        let block_comm_str = std::str::from_utf8(block_comm_bytes).unwrap();
        let tagged = TaggedBase64::parse(&block_comm_str).unwrap();
        let block_comm: BlockMerkleCommitment = tagged.try_into().unwrap();
        let mut block_comm_root_bytes = vec![];
        block_comm
            .serialize_compressed(&mut block_comm_root_bytes)
            .unwrap();
        let field_bytes = hash_bytes_to_field(&block_comm_root_bytes).unwrap();
        let circuit_block_u256 = field_to_u256(field_bytes);
        let mut circuit_block_bytes: Vec<u8> = vec![0; 32];
        circuit_block_u256.to_little_endian(&mut circuit_block_bytes);
        verify_merkle_proof_helper(
            proof_bytes,
            header_bytes,
            block_comm_bytes,
            &circuit_block_bytes,
        );
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
