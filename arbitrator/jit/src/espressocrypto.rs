use crate::caller_env::JitEnv;
use crate::machine::{MaybeEscape, WasmEnvMut};
use caller_env::{GuestPtr, MemAccess};
use espresso_crypto_helper::{verify_merkle_proof_helper, verify_namespace_helper};

pub fn verify_namespace(
    mut env: WasmEnvMut,
    namespace: u64,
    proof_ptr: GuestPtr,
    proof_len: u64,
    payload_comm_ptr: GuestPtr,
    payload_comm_len: u64,
    ns_table_ptr: GuestPtr,
    ns_table_len: u64,
    txs_comm_ptr: GuestPtr,
    txs_comm_len: u64,
    common_data_ptr: GuestPtr,
    common_data_len: u64,
) -> MaybeEscape {
    let (mem, _exec) = env.jit_env();

    let proof_bytes = mem.read_slice(proof_ptr, proof_len as usize);
    let payload_comm_bytes = mem.read_slice(payload_comm_ptr, payload_comm_len as usize);
    let ns_table_bytes = mem.read_slice(ns_table_ptr, ns_table_len as usize);
    let txs_comm_bytes = mem.read_slice(txs_comm_ptr, txs_comm_len as usize);
    let common_data_bytes = mem.read_slice(common_data_ptr, common_data_len as usize);

    Ok(verify_namespace_helper(
        namespace,
        &proof_bytes,
        &payload_comm_bytes,
        &ns_table_bytes,
        &txs_comm_bytes,
        &common_data_bytes,
    ))
}

pub fn verify_merkle_proof(
    mut env: WasmEnvMut,
    proof_ptr: GuestPtr,
    proof_len: u64,
    header_ptr: GuestPtr,
    header_len: u64,
    block_comm_ptr: GuestPtr,
    block_comm_len: u64,
    circuit_ptr: GuestPtr,
    circuit_len: u64,
) -> MaybeEscape {
    let (mem, _exec) = env.jit_env();

    let proof_bytes = mem.read_slice(proof_ptr, proof_len as usize);
    let header_bytes = mem.read_slice(header_ptr, header_len as usize);
    let block_comm_bytes = mem.read_slice(block_comm_ptr, block_comm_len as usize);
    let circuit_comm_bytes = mem.read_slice(circuit_ptr, circuit_len as usize);

    Ok(verify_merkle_proof_helper(
        &proof_bytes,
        &header_bytes,
        &block_comm_bytes,
        &circuit_comm_bytes,
    ))
}
