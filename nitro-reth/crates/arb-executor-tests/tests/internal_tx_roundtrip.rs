//! DB Compress/Decompress + 2718 round-trip fidelity for the StartBlock
//! InternalTx. The re-execute and sync paths read transactions back from the
//! reth database, so the InternalTx type byte must survive that round-trip;
//! otherwise the executor stops routing it through the ArbOS handler.

use alloy_consensus::Transaction as _;
use alloy_eips::{
    eip2718::{Decodable2718, Encodable2718},
    Typed2718,
};
use alloy_primitives::{Signature, U256};
use arb_alloy_consensus::tx::ArbInternalTx;
use arb_primitives::{signed_tx::ArbTypedTransaction, ArbTransactionSigned};
use arbos::internal_tx::encode_start_block;
use reth_db_api::table::{Compress, Decompress};

fn internal_tx() -> ArbTransactionSigned {
    let calldata = encode_start_block(U256::ZERO, 0xa606cf, 269_589_709, 1);
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(421614u64),
            data: calldata.into(),
        }),
        Signature::new(U256::ZERO, U256::ZERO, false),
    )
}

#[test]
fn internal_tx_survives_db_roundtrip() {
    let tx = internal_tx();
    assert_eq!(tx.ty(), 0x6a, "fresh tx type");

    let mut buf = Vec::new();
    tx.compress_to_buf(&mut buf);
    let decoded = ArbTransactionSigned::decompress(&buf).expect("decompress");
    eprintln!("after DB round-trip: ty=0x{:02x}", decoded.ty());
    assert_eq!(
        decoded.ty(),
        0x6a,
        "DB round-trip lost the InternalTx type byte"
    );
    assert_eq!(
        decoded.input().len(),
        tx.input().len(),
        "calldata length changed across DB round-trip"
    );
}

#[test]
fn internal_tx_survives_2718_roundtrip() {
    let tx = internal_tx();
    let bytes = tx.encoded_2718();
    eprintln!(
        "encoded_2718 first byte = 0x{:02x}, len={}",
        bytes[0],
        bytes.len()
    );
    let decoded = ArbTransactionSigned::decode_2718(&mut bytes.as_slice()).expect("decode_2718");
    eprintln!("after decode_2718: ty=0x{:02x}", decoded.ty());
    assert_eq!(
        decoded.ty(),
        0x6a,
        "2718 round-trip lost the InternalTx type"
    );
}
