use go_abi::*;
use vid_helper::verify_namespace_helper;

#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbvid_verifyNamespace(sp: GoStack) {
    let namespace = sp.read_u64(0);
    let proof_buf_ptr = sp.read_u64(1);
    let proof_buf_len = sp.read_u64(2);
    let payload_comm_buf_ptr = sp.read_u64(4);
    let payload_comm_buf_len = sp.read_u64(5);
    let ns_table_bytes_ptr = sp.read_u64(7);
    let ns_table_bytes_len = sp.read_u64(8);
    let txs_comm_ptr = sp.read_u64(10);
    let txs_comm_len = sp.read_u64(11);

    let proof_bytes = read_slice(proof_buf_ptr, proof_buf_len);
    let payload_comm_bytes = read_slice(payload_comm_buf_ptr, payload_comm_buf_len);
    let tx_comm_bytes = read_slice(txs_comm_ptr, txs_comm_len);
    let ns_table_bytes = read_slice(ns_table_bytes_ptr, ns_table_bytes_len);

    verify_namespace_helper(namespace, &proof_bytes, &payload_comm_bytes, &ns_table_bytes, &tx_comm_bytes)
}
