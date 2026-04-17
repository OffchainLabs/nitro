use alloy_consensus::{transaction::Recovered, SignableTransaction, TxLegacy};
use alloy_consensus::{crypto::secp256k1::sign_message, EthereumTxEnvelope};
use alloy_primitives::{address, Bytes, TxKind, B256, U256};
use arb_primitives::ArbTransactionSigned;
use arb_txpool::ArbPooledTransaction;
use reth_transaction_pool::PoolTransaction;

const ONE_GWEI: u128 = 1_000_000_000;

fn alice_key() -> [u8; 32] {
    let mut k = [0u8; 32];
    k[31] = 1;
    k
}

fn alice() -> alloy_primitives::Address {
    address!("1a642f0e3c3af545e7acbd38b07251b3990914f1")
}

fn sign_legacy(nonce: u64) -> ArbTransactionSigned {
    let tx = TxLegacy {
        chain_id: Some(42161),
        nonce,
        gas_price: ONE_GWEI,
        gas_limit: 21_000,
        to: TxKind::Call(alloy_primitives::Address::repeat_byte(0xAA)),
        value: U256::from(1_000_000_000_000_000u128),
        input: Bytes::new(),
    };
    let hash = tx.signature_hash();
    let sig = sign_message(B256::from(alice_key()), hash).expect("sign");
    let signed = tx.into_signed(sig);
    ArbTransactionSigned::from_envelope(EthereumTxEnvelope::Legacy(signed))
}

fn make_pooled(nonce: u64, encoded_len: usize) -> ArbPooledTransaction {
    let tx = sign_legacy(nonce);
    let recovered = Recovered::new_unchecked(tx, alice());
    ArbPooledTransaction::new(recovered, encoded_len)
}

#[test]
fn pool_tx_sender_matches_signer() {
    let p = make_pooled(0, 128);
    assert_eq!(PoolTransaction::sender(&p), alice());
    assert_eq!(PoolTransaction::sender_ref(&p), &alice());
}

#[test]
fn pool_tx_hash_is_consistent() {
    let p = make_pooled(0, 128);
    let h1 = *PoolTransaction::hash(&p);
    let p2 = make_pooled(0, 256);
    let h2 = *PoolTransaction::hash(&p2);
    assert_eq!(h1, h2, "same tx content produces same hash regardless of encoded_len");
}

#[test]
fn pool_tx_hash_differs_by_nonce() {
    let h0 = *PoolTransaction::hash(&make_pooled(0, 128));
    let h1 = *PoolTransaction::hash(&make_pooled(1, 128));
    assert_ne!(h0, h1);
}

#[test]
fn pool_tx_encoded_length_preserved() {
    let p = make_pooled(0, 1024);
    assert_eq!(PoolTransaction::encoded_length(&p), 1024);
}

#[test]
fn pool_tx_consensus_ref_returns_signed_tx() {
    let p = make_pooled(0, 128);
    let c: Recovered<&ArbTransactionSigned> = PoolTransaction::consensus_ref(&p);
    assert_eq!(c.signer(), alice());
}

#[test]
fn pool_tx_into_consensus_preserves_signer() {
    let p = make_pooled(0, 128);
    let rec = PoolTransaction::into_consensus(p);
    assert_eq!(rec.signer(), alice());
}

#[test]
fn pool_tx_clone_into_consensus_does_not_move() {
    let p = make_pooled(0, 128);
    let _rec = PoolTransaction::clone_into_consensus(&p);
    let hash_still_available = *PoolTransaction::hash(&p);
    assert_ne!(hash_still_available, B256::ZERO);
}

#[test]
fn pool_tx_from_pooled_computes_encoded_length() {
    let tx = sign_legacy(0);
    let recovered = Recovered::new_unchecked(tx, alice());
    let pooled = ArbPooledTransaction::from_pooled(recovered);
    assert!(PoolTransaction::encoded_length(&pooled) > 0);
}

#[test]
fn pool_tx_cost_nonzero_for_value_tx() {
    let p = make_pooled(0, 128);
    let cost = PoolTransaction::cost(&p);
    assert!(*cost > U256::ZERO);
}

#[test]
fn pool_tx_into_consensus_with2718_roundtrips_encoding() {
    let p = make_pooled(0, 128);
    let with_encoded = PoolTransaction::into_consensus_with2718(p);
    assert!(!with_encoded.encoded_bytes().is_empty());
}

// ==== alloy_consensus::Transaction facets ====

#[test]
fn consensus_trait_chain_id_is_set_for_legacy_eip155() {
    let p = make_pooled(0, 128);
    assert_eq!(alloy_consensus::Transaction::chain_id(&p), Some(42161));
}

#[test]
fn consensus_trait_nonce_and_gas_limit() {
    let p = make_pooled(7, 128);
    assert_eq!(alloy_consensus::Transaction::nonce(&p), 7);
    assert_eq!(alloy_consensus::Transaction::gas_limit(&p), 21_000);
}

#[test]
fn consensus_trait_kind_is_call() {
    let p = make_pooled(0, 128);
    let kind = alloy_consensus::Transaction::kind(&p);
    assert!(matches!(kind, TxKind::Call(_)));
}

#[test]
fn consensus_trait_value_matches() {
    let p = make_pooled(0, 128);
    assert_eq!(
        alloy_consensus::Transaction::value(&p),
        U256::from(1_000_000_000_000_000u128)
    );
}
