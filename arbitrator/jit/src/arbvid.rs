// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{gostack::GoStack, machine::WasmEnvMut};
use vid_helper::verify_namespace_helper;

pub fn verify_namespace(mut env: WasmEnvMut, sp: u32) {
    let (sp, _) = GoStack::new(sp, &mut env);

    let namespace = sp.read_u64(0);
    let proof_ptr = sp.read_u64(1);
    let proof_len = sp.read_u64(2);
    let payload_comm_ptr = sp.read_u64(4);
    let payload_comm_len = sp.read_u64(5);
    let ns_table_bytes_ptr = sp.read_u64(7);
    let ns_table_bytes_len = sp.read_u64(8);
    let txs_comm_ptr = sp.read_u64(10);
    let txs_comm_len = sp.read_u64(11);

    let proof_bytes = sp.read_slice(proof_ptr, proof_len);
    let payload_comm_bytes = sp.read_slice(payload_comm_ptr, payload_comm_len);
    let ns_table_bytes = sp.read_slice(ns_table_bytes_ptr, ns_table_bytes_len);
    let txs_comm_bytes = sp.read_slice(txs_comm_ptr, txs_comm_len);

    verify_namespace_helper(namespace, &proof_bytes, &payload_comm_bytes, &ns_table_bytes, &txs_comm_bytes)

}

