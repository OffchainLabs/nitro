// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{gostack::GoStack, machine::WasmEnvMut};
use ark_bls12_381::Bls12_381;
use ark_serialize::CanonicalDeserialize;
use jf_primitives::{
    pcs::prelude::UnivariateUniversalParams,
    vid::{advz::Advz, VidScheme as VidSchemeTrait},
};
use lazy_static::lazy_static;
use sequencer::{block::entry::TxTableEntryWord, Transaction};
use sequencer::block::{payload::NamespaceProof, tables::NameSpaceTable};
use tagged_base64::TaggedBase64;
use sha2::{Sha256, Digest};

pub type VidScheme = Advz<Bls12_381, sha2::Sha256>;

lazy_static! {
    // Initialize the byte array from JSON content
    static ref SRS_VEC: Vec<u8> = {
        let json_content = include_str!("../../../config/vid_srs.json");
        serde_json::from_str(json_content).expect("Failed to deserialize")
    };
}

pub fn verify_namespace(mut env: WasmEnvMut, sp: u32) {
    let (sp, _) = GoStack::new(sp, &mut env);

    let namespace = sp.read_u64(0);
    let proof_buf_ptr = sp.read_u64(1);
    let proof_buf_len = sp.read_u64(2);
    let comm_buf_ptr = sp.read_u64(4);
    let comm_buf_len = sp.read_u64(5);
    let ns_table_bytes_ptr = sp.read_u64(7);
    let ns_table_bytes_len = sp.read_u64(8);
    let txs_buf_ptr = sp.read_u64(10);
    let txs_buf_len = sp.read_u64(11);

    let proof_bytes = sp.read_slice(proof_buf_ptr, proof_buf_len);
    let comm_bytes = sp.read_slice(comm_buf_ptr, comm_buf_len);
    let txs_bytes = sp.read_slice(txs_buf_ptr, txs_buf_len);
    let ns_table_bytes = sp.read_slice(ns_table_bytes_ptr, ns_table_bytes_len);

    verify_namespace_helper(namespace, &proof_bytes, &comm_bytes, &ns_table_bytes, &txs_bytes)

}


// Helper function to verify a VID namespace proof that takes the byte representations of the proof,
// namespace table, and commitment string.
//
// proof_bytes: Byte representation of a JSON NamespaceProof string
// commit_bytes: Byte representation of a TaggedBase64 payload commitment string
// ns_table_bytes: Raw bytes of the namespace table
fn verify_namespace_helper(namespace: u64, proof_bytes: &[u8], commit_bytes: &[u8], ns_table_bytes: &[u8], tx_comm_bytes: &[u8]) {
    // Create proof and commit strings
    let proof_str = std::str::from_utf8(proof_bytes).unwrap();
    let commit_str = std::str::from_utf8(commit_bytes).unwrap();
    let txn_comm_str =std::str::from_utf8(tx_comm_bytes).unwrap();

    let proof: NamespaceProof = serde_json::from_str(proof_str).unwrap();
    let ns_table = NameSpaceTable::<TxTableEntryWord>::from_vec(ns_table_bytes.to_vec());
    let tagged = TaggedBase64::parse(&commit_str).unwrap();
    let commit: <VidScheme as VidSchemeTrait>::Commit = tagged.try_into().unwrap();

    let advz: Advz<Bls12_381, sha2::Sha256>;
    let srs = UnivariateUniversalParams::<Bls12_381>::deserialize_compressed(&**SRS_VEC).unwrap();
    let (payload_chunk_size, num_storage_nodes) = (8, 10);
    advz = Advz::new(payload_chunk_size, num_storage_nodes, srs).unwrap();
    let (txns, ns) = proof.verify(&advz, &commit, &ns_table).unwrap();

    let txns_comm = hash_txns(namespace, &txns);
    
    assert!(ns == namespace.into());
    assert!(txns_comm == txn_comm_str);
}

fn hash_txns(namespace: u64, txns: &[Transaction]) -> String {
    let mut hasher = Sha256::new();
    //hasher.update(namespace);
    for txn in txns {
        hasher.update(&txn.payload());
    }
    let hash_result = hasher.finalize();
    format!("{:x}", hash_result)
}

#[test]
fn test_verify_namespace_helper() {
    let proof_bytes = b"{\"NonExistence\":{\"ns_id\":0}}";
    let commit_bytes = b"HASH~1yS-KEtL3oDZDBJdsW51Pd7zywIiHesBZsTbpOzrxOfu";
    let txn_comm_bytes = b"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";
    let ns_table_bytes = &[0,0,0,0]; 
    verify_namespace_helper(0, proof_bytes, commit_bytes, ns_table_bytes, txn_comm_bytes);
}
