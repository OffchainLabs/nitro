use ark_ff::{BigInteger, PrimeField};
use ark_serialize::CanonicalSerialize;
use committable::{Commitment, Committable};
use espresso_types::{
    BlockMerkleCommitment, BlockMerkleTree, Header, NsProof, NsTable, Transaction,
};
use ethers_core::types::U256;
use hotshot_types::vid::{VidCommitment, VidCommon};
use jf_crhf::CRHF;
use jf_merkle_tree::prelude::{
    MerkleCommitment, MerkleNode, MerkleProof, MerkleTreeScheme, Sha3Node,
};
use jf_rescue::{crhf::VariableLengthRescueCRHF, RescueError};
use sha2::{Digest, Sha256};
use tagged_base64::TaggedBase64;

pub type Proof = Vec<MerkleNode<Commitment<Header>, u64, Sha3Node>>;
pub type CircuitField = ark_ed_on_bn254::Fq;

// Helper function to verify a block merkle proof.
// proof_bytes: Byte representation of a block merkle proof.
// root_bytes: Byte representation of a Sha3Node merkle root.
// header_bytes: Byte representation of the HotShot header being validated as a Merkle leaf.
// circuit_block_bytes: Circuit representation of the HotShot header commitment returned by the light client contract.
#[no_mangle]
pub extern "C" fn verify_merkle_proof_helper(
    proof_ptr: *const u8,
    proof_len: usize,
    header_ptr: *const u8,
    header_len: usize,
    block_comm_ptr: *const u8,
    block_comm_len: usize,
    circuit_block_ptr: *const u8,
    circuit_block_len: usize,
) -> bool {
    let proof_bytes = unsafe { std::slice::from_raw_parts(proof_ptr, proof_len) };
    let header_bytes = unsafe { std::slice::from_raw_parts(header_ptr, header_len) };
    let block_comm_bytes = unsafe { std::slice::from_raw_parts(block_comm_ptr, block_comm_len) };
    let circuit_block_bytes =
        unsafe { std::slice::from_raw_parts(circuit_block_ptr, circuit_block_len) };

    let block_comm_str = std::str::from_utf8(block_comm_bytes).unwrap();
    let tagged = TaggedBase64::parse(&block_comm_str).unwrap();
    let block_comm: BlockMerkleCommitment = tagged.try_into().unwrap();

    let proof: Proof = serde_json::from_slice(proof_bytes).unwrap();
    let header: Header = serde_json::from_slice(header_bytes).unwrap();
    let header_comm: Commitment<Header> = header.commit();

    let proof = MerkleProof::new(header.height(), proof.to_vec());
    let proved_comm = proof.elem().unwrap().clone();
    BlockMerkleTree::verify(block_comm.digest(), header.height(), proof)
        .unwrap()
        .unwrap();

    let mut block_comm_root_bytes = vec![];
    block_comm
        .serialize_compressed(&mut block_comm_root_bytes)
        .unwrap();
    let field_bytes = hash_bytes_to_field(&block_comm_root_bytes).unwrap();
    let local_block_comm_u256 = field_to_u256(field_bytes);
    let circuit_block_comm_u256 = U256::from_little_endian(circuit_block_bytes);

    if (proved_comm == header_comm) && (local_block_comm_u256 == circuit_block_comm_u256) {
        return true;
    }
    return false;
}

// Helper function to verify a VID namespace proof that takes the byte representations of the proof,
// namespace table, and commitment string.
//
// proof_bytes: Byte representation of a JSON NamespaceProof string.
// commit_bytes: Byte representation of a TaggedBase64 payload commitment string.
// ns_table_bytes: Raw bytes of the namespace table.
// tx_comm_bytes: Byte representation of a hex encoded Sha256 digest that the transaction set commits to.
#[no_mangle]
pub extern "C" fn verify_namespace_helper(
    namespace: u64,
    proof_ptr: *const u8,
    proof_len: usize,
    commit_ptr: *const u8,
    commit_len: usize,
    ns_table_ptr: *const u8,
    ns_table_len: usize,
    tx_comm_ptr: *const u8,
    tx_comm_len: usize,
    common_data_ptr: *const u8,
    common_data_len: usize,
) -> bool {
    let ns_table_bytes = unsafe { std::slice::from_raw_parts(ns_table_ptr, ns_table_len) };
    let proof_bytes = unsafe { std::slice::from_raw_parts(proof_ptr, proof_len) };
    let commit_bytes = unsafe { std::slice::from_raw_parts(commit_ptr, commit_len) };
    let tx_comm_bytes = unsafe { std::slice::from_raw_parts(tx_comm_ptr, tx_comm_len) };
    let common_data_bytes = unsafe { std::slice::from_raw_parts(common_data_ptr, common_data_len) };

    let commit_str = std::str::from_utf8(commit_bytes).unwrap();
    let txn_comm_str = std::str::from_utf8(tx_comm_bytes).unwrap();

    let proof: NsProof = serde_json::from_slice(proof_bytes).unwrap();
    let ns_table: NsTable = NsTable::from_bytes_unchecked(ns_table_bytes);
    let tagged = TaggedBase64::parse(&commit_str).unwrap();
    let commit: VidCommitment = tagged.try_into().unwrap();
    let vid_common: VidCommon = serde_json::from_slice(common_data_bytes).unwrap();

    let (txns, ns) = proof.verify(&ns_table, &commit, &vid_common).unwrap();

    let namespace: u32 = namespace.try_into().unwrap();
    let txns_comm = hash_txns(namespace, &txns);

    if (ns == namespace.into()) && (txns_comm == txn_comm_str) {
        return true;
    }
    return false;
}

// TODO: Use Commit trait: https://github.com/EspressoSystems/nitro-espresso-integration/issues/88
fn hash_txns(namespace: u32, txns: &[Transaction]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(namespace.to_le_bytes());
    for txn in txns {
        hasher.update(txn.payload());
    }
    let hash_result = hasher.finalize();
    format!("{:x}", hash_result)
}

fn hash_bytes_to_field(bytes: &[u8]) -> Result<CircuitField, RescueError> {
    // make sure that `mod_order` won't happen.
    let bytes_len = ((<CircuitField as PrimeField>::MODULUS_BIT_SIZE + 7) / 8 - 1) as usize;
    let elem = bytes
        .chunks(bytes_len)
        .map(CircuitField::from_le_bytes_mod_order)
        .collect::<Vec<_>>();
    Ok(VariableLengthRescueCRHF::<_, 1>::evaluate(elem)?[0])
}

pub fn field_to_u256<F: PrimeField>(f: F) -> U256 {
    if F::MODULUS_BIT_SIZE > 256 {
        panic!("Shouldn't convert a >256-bit field to U256");
    }
    U256::from_little_endian(&f.into_bigint().to_bytes_le())
}
