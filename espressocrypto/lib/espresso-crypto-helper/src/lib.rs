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

macro_rules! handle_result {
    ($result:expr) => {
        match $result {
            Ok(value) => value,
            Err(_) => return false,
        }
    };
}

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
    let proof_bytes = handle_result!(slice_from_raw_parts(proof_ptr, proof_len));
    let header_bytes = handle_result!(slice_from_raw_parts(header_ptr, header_len));
    let block_comm_bytes = handle_result!(slice_from_raw_parts(block_comm_ptr, block_comm_len));
    let circuit_block_bytes =
        handle_result!(slice_from_raw_parts(circuit_block_ptr, circuit_block_len));

    let block_comm_str = handle_result!(std::str::from_utf8(block_comm_bytes));
    let tagged = handle_result!(TaggedBase64::parse(&block_comm_str));
    let block_comm: BlockMerkleCommitment = handle_result!(tagged.try_into());

    let proof: Proof = handle_result!(serde_json::from_slice(proof_bytes));
    let header: Header = handle_result!(serde_json::from_slice(header_bytes));
    let header_comm: Commitment<Header> = header.commit();

    let proof = MerkleProof::new(header.height(), proof.to_vec());
    let proved_comm = if let Some(p) = proof.elem() {
        p.clone()
    } else {
        return false;
    };
    handle_result!(handle_result!(BlockMerkleTree::verify(
        block_comm.digest(),
        header.height(),
        proof
    )));

    let mut block_comm_root_bytes = vec![];
    handle_result!(block_comm.serialize_compressed(&mut block_comm_root_bytes));
    let field_bytes = handle_result!(hash_bytes_to_field(&block_comm_root_bytes));
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
    let ns_table_bytes = handle_result!(slice_from_raw_parts(ns_table_ptr, ns_table_len));
    let proof_bytes = handle_result!(slice_from_raw_parts(proof_ptr, proof_len));
    let commit_bytes = handle_result!(slice_from_raw_parts(commit_ptr, commit_len));
    let tx_comm_bytes = handle_result!(slice_from_raw_parts(tx_comm_ptr, tx_comm_len));
    let common_data_bytes = handle_result!(slice_from_raw_parts(common_data_ptr, common_data_len));

    let commit_str = handle_result!(std::str::from_utf8(commit_bytes));
    let txn_comm_str = handle_result!(std::str::from_utf8(tx_comm_bytes));

    let proof: NsProof = handle_result!(serde_json::from_slice(proof_bytes));
    let ns_table: NsTable = NsTable::from_bytes_unchecked(ns_table_bytes);
    let tagged = handle_result!(TaggedBase64::parse(&commit_str));
    let commit: VidCommitment = handle_result!(tagged.try_into());
    let vid_common: VidCommon = handle_result!(serde_json::from_slice(common_data_bytes));

    let (txns, ns) = handle_result!(proof.verify(&ns_table, &commit, &vid_common).ok_or(()));

    let namespace: u32 = handle_result!(namespace.try_into());
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

fn slice_from_raw_parts<'a>(ptr: *const u8, len: usize) -> Result<&'a [u8], ()> {
    if ptr.is_null() {
        return Err(());
    }
    if !ptr.is_aligned() {
        return Err(());
    }
    // Check if the range overflows
    if usize::MAX - (ptr as usize) < len {
        return Err(());
    }
    Ok(unsafe { std::slice::from_raw_parts(ptr, len) })
}
