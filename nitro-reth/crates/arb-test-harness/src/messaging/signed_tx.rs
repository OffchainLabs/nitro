use alloy_consensus::{
    crypto::secp256k1::sign_message, EthereumTxEnvelope, SignableTransaction, TxEip1559, TxEip2930,
    TxLegacy,
};
use alloy_eips::{eip2718::Encodable2718, eip2930::AccessList, eip2930::AccessListItem};
use alloy_primitives::{keccak256, Address, Bytes, B256, U256};

use crate::{
    error::HarnessError,
    messaging::{kinds, L1Message, L1MessageHeader, MessageBuilder},
};

/// Selects which Ethereum tx envelope is wrapped inside the L2 SignedTx
/// sub-message.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum L2TxKind {
    Legacy,
    Eip2930,
    Eip1559,
}

/// Builds an L1 message of kind 3 (L2Message) carrying a SignedTx sub-message
/// (sub-kind 0x04) whose payload is the EIP-2718 encoding of an ECDSA-signed
/// Ethereum transaction.
#[derive(Debug, Clone)]
pub struct SignedL2TxBuilder {
    pub chain_id: u64,
    pub nonce: u64,
    /// `None` builds a CREATE transaction.
    pub to: Option<Address>,
    pub value: U256,
    pub data: Bytes,
    pub gas_limit: u64,
    /// Used for legacy and EIP-2930 envelopes.
    pub gas_price: u128,
    /// Used for EIP-1559 envelopes.
    pub max_fee_per_gas: u128,
    /// Used for EIP-1559 envelopes.
    pub max_priority_fee_per_gas: u128,
    /// Used for EIP-2930 and EIP-1559 envelopes.
    pub access_list: Vec<(Address, Vec<B256>)>,
    pub kind: L2TxKind,
    /// 32-byte secp256k1 secret key.
    pub signing_key: B256,
    /// L1 block number recorded in the L1 message header.
    pub l1_block_number: u64,
    /// L1 timestamp recorded in the L1 message header.
    pub timestamp: u64,
    /// Optional request id; SignedTx messages typically leave this `None`.
    pub request_id: Option<B256>,
    /// Sender address recorded in the L1 message header. Sequencer-posted
    /// SignedTx messages set this to the sequencer alias address. The signer
    /// of the inner tx is recovered from its signature, not from this field.
    pub sender: Address,
    /// L1 base fee recorded in the L1 message header.
    pub base_fee_l1: u64,
}

impl SignedL2TxBuilder {
    /// Returns the EOA address that owns `signing_key`.
    pub fn sender(&self) -> Address {
        derive_address(self.signing_key)
    }

    /// Constructs the EIP-2718 encoding of the signed inner Ethereum tx.
    pub fn signed_envelope_2718(&self) -> crate::Result<Vec<u8>> {
        let envelope = self.build_envelope()?;
        let mut buf = Vec::new();
        envelope.encode_2718(&mut buf);
        Ok(buf)
    }

    /// Returns the bare L2-message payload `[0x04 || rlp(signed_tx)]` (without
    /// the outer L1 header).
    pub fn encode_body(&self) -> crate::Result<Vec<u8>> {
        let inner = self.signed_envelope_2718()?;
        let mut out = Vec::with_capacity(1 + inner.len());
        out.push(kinds::KIND_SIGNED_L2_TX);
        out.extend_from_slice(&inner);
        Ok(out)
    }

    fn build_envelope(&self) -> crate::Result<EthereumTxEnvelope<alloy_consensus::TxEip4844>> {
        let to = match self.to {
            Some(a) => alloy_primitives::TxKind::Call(a),
            None => alloy_primitives::TxKind::Create,
        };
        match self.kind {
            L2TxKind::Legacy => {
                let tx = TxLegacy {
                    chain_id: Some(self.chain_id),
                    nonce: self.nonce,
                    gas_price: self.gas_price,
                    gas_limit: self.gas_limit,
                    to,
                    value: self.value,
                    input: self.data.clone(),
                };
                let sig_hash = tx.signature_hash();
                let sig = sign_message(self.signing_key, sig_hash)
                    .map_err(|e| HarnessError::Invalid(format!("legacy sign failed: {e}")))?;
                Ok(EthereumTxEnvelope::Legacy(tx.into_signed(sig)))
            }
            L2TxKind::Eip2930 => {
                let tx = TxEip2930 {
                    chain_id: self.chain_id,
                    nonce: self.nonce,
                    gas_price: self.gas_price,
                    gas_limit: self.gas_limit,
                    to,
                    value: self.value,
                    access_list: build_access_list(&self.access_list),
                    input: self.data.clone(),
                };
                let sig_hash = tx.signature_hash();
                let sig = sign_message(self.signing_key, sig_hash)
                    .map_err(|e| HarnessError::Invalid(format!("eip2930 sign failed: {e}")))?;
                Ok(EthereumTxEnvelope::Eip2930(tx.into_signed(sig)))
            }
            L2TxKind::Eip1559 => {
                let tx = TxEip1559 {
                    chain_id: self.chain_id,
                    nonce: self.nonce,
                    gas_limit: self.gas_limit,
                    max_fee_per_gas: self.max_fee_per_gas,
                    max_priority_fee_per_gas: self.max_priority_fee_per_gas,
                    to,
                    value: self.value,
                    access_list: build_access_list(&self.access_list),
                    input: self.data.clone(),
                };
                let sig_hash = tx.signature_hash();
                let sig = sign_message(self.signing_key, sig_hash)
                    .map_err(|e| HarnessError::Invalid(format!("eip1559 sign failed: {e}")))?;
                Ok(EthereumTxEnvelope::Eip1559(tx.into_signed(sig)))
            }
        }
    }
}

impl MessageBuilder for SignedL2TxBuilder {
    fn build(&self) -> crate::Result<L1Message> {
        let body = self.encode_body()?;
        Ok(L1Message {
            header: L1MessageHeader {
                kind: kinds::KIND_L2_MESSAGE,
                sender: self.sender,
                block_number: self.l1_block_number,
                timestamp: self.timestamp,
                request_id: self.request_id,
                base_fee_l1: self.base_fee_l1,
            },
            l2_msg: crate::messaging::b64_l2_msg(&body.into()),
        })
    }
}

fn build_access_list(items: &[(Address, Vec<B256>)]) -> AccessList {
    AccessList(
        items
            .iter()
            .map(|(addr, slots)| AccessListItem {
                address: *addr,
                storage_keys: slots.clone(),
            })
            .collect(),
    )
}

/// Derives the Ethereum address from a secp256k1 secret key by hashing the
/// uncompressed public key (without the 0x04 prefix) with keccak256 and
/// taking the last 20 bytes.
pub fn derive_address(sk: B256) -> Address {
    let signing_key =
        k256::ecdsa::SigningKey::from_slice(sk.as_slice()).expect("32-byte secret");
    let verifying = *signing_key.verifying_key();
    let encoded = verifying.to_encoded_point(false);
    let pubkey = &encoded.as_bytes()[1..];
    let hash = keccak256(pubkey);
    Address::from_slice(&hash[12..])
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::messaging::test_support::{decode_body, round_trip};
    use alloy_consensus::transaction::SignerRecoverable;
    use alloy_eips::eip2718::Decodable2718;
    use alloy_primitives::{address, b256, hex};
    use arb_primitives::signed_tx::ArbTransactionSigned;
    use arbos::parse_l2::{parse_l2_transactions, ParsedTransaction};

    /// Hardhat default account #0 (private key well-known across the ecosystem).
    fn hardhat_key_0() -> B256 {
        b256!("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
    }

    fn hardhat_addr_0() -> Address {
        address!("f39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
    }

    fn legacy_builder(key: B256) -> SignedL2TxBuilder {
        SignedL2TxBuilder {
            chain_id: 421_614,
            nonce: 7,
            to: Some(address!("00000000000000000000000000000000000000bb")),
            value: U256::from(123_456u64),
            data: Bytes::from_static(&[0x10, 0x20]),
            gas_limit: 100_000,
            gas_price: 1_000_000_000,
            max_fee_per_gas: 0,
            max_priority_fee_per_gas: 0,
            access_list: Vec::new(),
            kind: L2TxKind::Legacy,
            signing_key: key,
            l1_block_number: 50,
            timestamp: 1_700_000_000,
            request_id: None,
            sender: address!("a4b000000000000000000073657175656e636572"),
            base_fee_l1: 0,
        }
    }

    fn eip2930_builder(key: B256) -> SignedL2TxBuilder {
        SignedL2TxBuilder {
            chain_id: 421_614,
            nonce: 9,
            to: Some(address!("00000000000000000000000000000000000000cc")),
            value: U256::from(42u64),
            data: Bytes::from_static(&[0xab]),
            gas_limit: 90_000,
            gas_price: 1_500_000_000,
            max_fee_per_gas: 0,
            max_priority_fee_per_gas: 0,
            access_list: vec![(
                address!("0000000000000000000000000000000000000abc"),
                vec![b256!(
                    "0000000000000000000000000000000000000000000000000000000000000001"
                )],
            )],
            kind: L2TxKind::Eip2930,
            signing_key: key,
            l1_block_number: 60,
            timestamp: 1_700_000_500,
            request_id: None,
            sender: address!("a4b000000000000000000073657175656e636572"),
            base_fee_l1: 0,
        }
    }

    fn eip1559_builder(key: B256) -> SignedL2TxBuilder {
        SignedL2TxBuilder {
            chain_id: 421_614,
            nonce: 1,
            to: None,
            value: U256::ZERO,
            data: Bytes::from_static(&[0x60, 0x05, 0x60, 0x00]),
            gas_limit: 250_000,
            gas_price: 0,
            max_fee_per_gas: 2_000_000_000,
            max_priority_fee_per_gas: 100_000_000,
            access_list: Vec::new(),
            kind: L2TxKind::Eip1559,
            signing_key: key,
            l1_block_number: 70,
            timestamp: 1_700_001_000,
            request_id: None,
            sender: address!("a4b000000000000000000073657175656e636572"),
            base_fee_l1: 0,
        }
    }

    #[test]
    fn sender_matches_hardhat_account_0() {
        assert_eq!(derive_address(hardhat_key_0()), hardhat_addr_0());
        let b = legacy_builder(hardhat_key_0());
        assert_eq!(b.sender(), hardhat_addr_0());
    }

    #[test]
    fn body_starts_with_signed_sub_kind() {
        let msg = legacy_builder(hardhat_key_0()).build().unwrap();
        let body = decode_body(&msg);
        assert_eq!(body[0], kinds::KIND_SIGNED_L2_TX);
        assert_eq!(msg.header.kind, kinds::KIND_L2_MESSAGE);
    }

    #[test]
    fn legacy_round_trips_through_arbos_parser() {
        let b = legacy_builder(hardhat_key_0());
        let msg = b.build().unwrap();
        let parsed = round_trip(&msg);
        let txs = parse_l2_transactions(
            parsed.header.kind,
            parsed.header.poster,
            &parsed.l2_msg,
            parsed.header.request_id,
            parsed.header.l1_base_fee,
            b.chain_id,
        )
        .unwrap();
        assert_eq!(txs.len(), 1);
        let rlp = match &txs[0] {
            ParsedTransaction::Signed(rlp) => rlp.clone(),
            other => panic!("unexpected parsed kind: {other:?}"),
        };
        let envelope = ArbTransactionSigned::decode_2718(&mut rlp.as_slice()).unwrap();
        let signer = envelope.recover_signer().unwrap();
        assert_eq!(signer, b.sender());
    }

    #[test]
    fn eip2930_round_trips_through_arbos_parser() {
        let b = eip2930_builder(hardhat_key_0());
        let msg = b.build().unwrap();
        let parsed = round_trip(&msg);
        let txs = parse_l2_transactions(
            parsed.header.kind,
            parsed.header.poster,
            &parsed.l2_msg,
            parsed.header.request_id,
            parsed.header.l1_base_fee,
            b.chain_id,
        )
        .unwrap();
        let rlp = match &txs[0] {
            ParsedTransaction::Signed(rlp) => rlp.clone(),
            other => panic!("unexpected parsed kind: {other:?}"),
        };
        let envelope = ArbTransactionSigned::decode_2718(&mut rlp.as_slice()).unwrap();
        assert_eq!(envelope.recover_signer().unwrap(), b.sender());
        // Also decode strictly via the alloy Ethereum envelope to check shape.
        let eth_env = EthereumTxEnvelope::<alloy_consensus::TxEip4844>::decode_2718(&mut rlp.as_slice())
            .unwrap();
        assert!(matches!(eth_env, EthereumTxEnvelope::Eip2930(_)));
    }

    #[test]
    fn eip1559_round_trips_through_arbos_parser_create() {
        let b = eip1559_builder(hardhat_key_0());
        let msg = b.build().unwrap();
        let parsed = round_trip(&msg);
        let txs = parse_l2_transactions(
            parsed.header.kind,
            parsed.header.poster,
            &parsed.l2_msg,
            parsed.header.request_id,
            parsed.header.l1_base_fee,
            b.chain_id,
        )
        .unwrap();
        let rlp = match &txs[0] {
            ParsedTransaction::Signed(rlp) => rlp.clone(),
            other => panic!("unexpected parsed kind: {other:?}"),
        };
        let eth_env =
            EthereumTxEnvelope::<alloy_consensus::TxEip4844>::decode_2718(&mut rlp.as_slice())
                .unwrap();
        assert!(matches!(eth_env, EthereumTxEnvelope::Eip1559(_)));
        let envelope = ArbTransactionSigned::decode_2718(&mut rlp.as_slice()).unwrap();
        assert_eq!(envelope.recover_signer().unwrap(), b.sender());
    }

    #[test]
    fn alloy_consensus_decodes_each_variant() {
        for kind in [L2TxKind::Legacy, L2TxKind::Eip2930, L2TxKind::Eip1559] {
            let mut b = legacy_builder(hardhat_key_0());
            b.kind = kind;
            let bytes = b.signed_envelope_2718().unwrap();
            let env = EthereumTxEnvelope::<alloy_consensus::TxEip4844>::decode_2718(
                &mut bytes.as_slice(),
            )
            .expect("alloy decodes each envelope");
            match (kind, &env) {
                (L2TxKind::Legacy, EthereumTxEnvelope::Legacy(_)) => {}
                (L2TxKind::Eip2930, EthereumTxEnvelope::Eip2930(_)) => {}
                (L2TxKind::Eip1559, EthereumTxEnvelope::Eip1559(_)) => {}
                _ => panic!("envelope variant did not match builder kind: {kind:?}"),
            }
        }
    }

    #[test]
    fn matches_known_fixture_envelope_shape() {
        // Decoded body of the kind=3 SignedTx message in
        // crates/arb-spec-tests/fixtures/execution/delayed_signed_tx_credits_recipient.json.
        // Layout: [0x04][0x02][rlp(eip1559_signed)].
        const FIXTURE_BODY_HEX: &str = concat!(
            "0402f86e83064aba8080840bebc20083018f4994000000000000000000000000",
            "00000000000000ff80840e5bbc11c080a065e8ebc276bd330e845a95ea3d5319",
            "7d6af7a67c179858838e2741c609d1117ca044c569ee9e457d4da76a7219a447",
            "2d282a8ceb0ae74d1fc1052440690bc60ff9",
        );
        let fixture_bytes = hex::decode(FIXTURE_BODY_HEX).unwrap();
        assert_eq!(fixture_bytes[0], kinds::KIND_SIGNED_L2_TX);
        assert_eq!(fixture_bytes[1], 0x02); // EIP-1559 type marker.

        // Re-decode the inner envelope using alloy and check chain id and shape.
        let inner = &fixture_bytes[1..];
        let env = EthereumTxEnvelope::<alloy_consensus::TxEip4844>::decode_2718(
            &mut inner.as_ref(),
        )
        .expect("alloy decodes fixture inner envelope");
        match env {
            EthereumTxEnvelope::Eip1559(signed) => {
                assert_eq!(signed.tx().chain_id, 412_346);
            }
            other => panic!("unexpected variant: {other:?}"),
        }

        // Build our own EIP-1559 envelope with the same wrapper byte and
        // top-level type marker so the structural shape matches.
        let mut b = legacy_builder(hardhat_key_0());
        b.chain_id = 412_346;
        b.kind = L2TxKind::Eip1559;
        b.max_fee_per_gas = 200_000_000;
        b.max_priority_fee_per_gas = 0;
        let body = b.encode_body().unwrap();
        assert_eq!(body[0], kinds::KIND_SIGNED_L2_TX);
        assert_eq!(body[1], 0x02);
        let env_ours = EthereumTxEnvelope::<alloy_consensus::TxEip4844>::decode_2718(
            &mut &body[1..],
        )
        .expect("alloy decodes our inner envelope");
        assert!(matches!(env_ours, EthereumTxEnvelope::Eip1559(_)));
    }

    #[test]
    fn json_round_trip_preserves_fields() {
        let msg = legacy_builder(hardhat_key_0()).build().unwrap();
        let v = serde_json::to_value(&msg).unwrap();
        let back: L1Message = serde_json::from_value(v).unwrap();
        assert_eq!(back.header.kind, kinds::KIND_L2_MESSAGE);
        assert!(back.header.request_id.is_none());
        assert_eq!(back.l2_msg, msg.l2_msg);
    }

    #[test]
    fn signature_is_deterministic_per_inputs() {
        let b1 = legacy_builder(hardhat_key_0());
        let b2 = legacy_builder(hardhat_key_0());
        assert_eq!(
            b1.signed_envelope_2718().unwrap(),
            b2.signed_envelope_2718().unwrap()
        );
    }

    #[test]
    fn recovered_signer_does_not_depend_on_header_sender() {
        let b = legacy_builder(hardhat_key_0());
        let bytes = b.signed_envelope_2718().unwrap();
        let envelope = ArbTransactionSigned::decode_2718(&mut bytes.as_slice()).unwrap();
        assert_eq!(envelope.recover_signer().unwrap(), b.sender());
        // Header.sender is the sequencer alias, not the EOA.
        let msg = b.build().unwrap();
        assert_eq!(
            msg.header.sender,
            address!("a4b000000000000000000073657175656e636572")
        );
    }
}
