use caller_env::{static_caller::STATIC_MEM, GuestPtr, MemAccess};
use espresso_crypto_helper::{verify_merkle_proof_helper, verify_namespace_helper};

#[no_mangle]
pub unsafe extern "C" fn espressocrypto__verifyNamespace(
    namespace: u64,
    proof_ptr: GuestPtr,
    proof_len: u64,
    payload_comm_ptr: GuestPtr,
    payload_comm_len: u64,
    ns_table_ptr: GuestPtr,
    ns_table_len: u64,
    txs_comm_ptr: GuestPtr,
    txs_comm_len: u64,
) {
    let proof_bytes = STATIC_MEM.read_slice(proof_ptr, proof_len as usize);
    let payload_comm_bytes = STATIC_MEM.read_slice(payload_comm_ptr, payload_comm_len as usize);
    let tx_comm_bytes = STATIC_MEM.read_slice(txs_comm_ptr, txs_comm_len as usize);
    let ns_table_bytes = STATIC_MEM.read_slice(ns_table_ptr, ns_table_len as usize);

    verify_namespace_helper(
        namespace,
        &proof_bytes,
        &payload_comm_bytes,
        &ns_table_bytes,
        &tx_comm_bytes,
    )
}

#[no_mangle]
pub unsafe extern "C" fn espressocrypto__verifyMerkleProof(
    proof_ptr: GuestPtr,
    proof_len: u64,
    header_ptr: GuestPtr,
    header_len: u64,
    block_comm_ptr: GuestPtr,
    block_comm_len: u64,
    circuit_ptr: GuestPtr,
    circuit_len: u64,
) {
    let proof_bytes = STATIC_MEM.read_slice(proof_ptr, proof_len as usize);
    let header_bytes = STATIC_MEM.read_slice(header_ptr, header_len as usize);
    let block_comm_bytes = STATIC_MEM.read_slice(block_comm_ptr, block_comm_len as usize);
    let circuit_comm_bytes = STATIC_MEM.read_slice(circuit_ptr, circuit_len as usize);

    verify_merkle_proof_helper(
        &proof_bytes,
        &header_bytes,
        &block_comm_bytes,
        &circuit_comm_bytes,
    )
}
