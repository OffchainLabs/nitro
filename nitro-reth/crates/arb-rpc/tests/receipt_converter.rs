//! Golden tests for [`arb_rpc::ArbReceiptConverter`].
//!
//! Locks the exact RPC receipt shape for each `ArbReceiptKind`. Fields
//! covered: envelope type (override for Arbitrum tx types ≥ 0x64),
//! `gasUsedForL1`, `l1BlockNumber` when a block is provided, `multiGasUsed`,
//! and `effectiveGasPrice` per tx type when CollectTips is set.

use alloy_consensus::{
    transaction::Recovered, Block, BlockBody, Header, Receipt as AlloyReceipt, TxLegacy,
};
use alloy_primitives::{address, Bytes, Log, Signature, TxKind, B256, U256};
use arb_alloy_consensus::tx::{ArbDepositTx, ArbInternalTx, ArbRetryTx, ArbSubmitRetryableTx};
use arb_primitives::{
    multigas::MultiGas, receipt::ArbDepositReceipt, ArbReceipt, ArbReceiptKind,
    ArbTransactionSigned, ArbTypedTransaction,
};
use arb_rpc::ArbReceiptConverter;
use reth_primitives_traits::{SealedBlock, TransactionMeta};
use reth_rpc_convert::transaction::{ConvertReceiptInput, ReceiptConverter};

const SIGNER: alloy_primitives::Address = address!("0000000000000000000000000000000000000aaa");

fn meta() -> TransactionMeta {
    TransactionMeta {
        tx_hash: B256::repeat_byte(0xAA),
        index: 0,
        block_hash: B256::repeat_byte(0xBB),
        block_number: 42,
        base_fee: Some(1_000_000_000),
        excess_blob_gas: None,
        timestamp: 1_700_000_000,
    }
}

fn legacy_tx() -> ArbTransactionSigned {
    let tx = TxLegacy {
        chain_id: Some(42161),
        nonce: 0,
        gas_price: 1_000_000_000,
        gas_limit: 21_000,
        to: TxKind::Call(address!("00000000000000000000000000000000000000bb")),
        value: U256::ZERO,
        input: Bytes::new(),
    };
    ArbTransactionSigned::new_unhashed(ArbTypedTransaction::Legacy(tx), Signature::test_signature())
}

fn deposit_tx() -> ArbTransactionSigned {
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Deposit(ArbDepositTx {
            chain_id: U256::from(42161u64),
            l1_request_id: B256::repeat_byte(0x42),
            from: SIGNER,
            to: address!("00000000000000000000000000000000000000bb"),
            value: U256::from(1_000u64),
        }),
        Signature::test_signature(),
    )
}

fn retry_tx() -> ArbTransactionSigned {
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Retry(ArbRetryTx {
            chain_id: U256::from(42161u64),
            nonce: 0,
            from: SIGNER,
            gas_fee_cap: U256::from(1_000_000_000u64),
            gas: 100_000,
            to: Some(address!("00000000000000000000000000000000000000bb")),
            value: U256::ZERO,
            data: Bytes::new(),
            ticket_id: B256::repeat_byte(0xAB),
            refund_to: SIGNER,
            max_refund: U256::ZERO,
            submission_fee_refund: U256::ZERO,
        }),
        Signature::test_signature(),
    )
}

fn submit_retryable_tx() -> ArbTransactionSigned {
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::SubmitRetryable(ArbSubmitRetryableTx {
            chain_id: U256::from(42161u64),
            request_id: B256::repeat_byte(0x12),
            from: SIGNER,
            l1_base_fee: U256::ZERO,
            deposit_value: U256::from(10u64).pow(U256::from(18u64)),
            gas_fee_cap: U256::from(1u64),
            gas: 100_000,
            retry_to: Some(address!("00000000000000000000000000000000000000bb")),
            retry_value: U256::ZERO,
            beneficiary: SIGNER,
            max_submission_fee: U256::ZERO,
            fee_refund_addr: SIGNER,
            retry_data: Bytes::new(),
        }),
        Signature::test_signature(),
    )
}

fn internal_tx() -> ArbTransactionSigned {
    ArbTransactionSigned::new_unhashed(
        ArbTypedTransaction::Internal(ArbInternalTx {
            chain_id: U256::from(42161u64),
            data: Bytes::new(),
        }),
        Signature::test_signature(),
    )
}

fn convert_one(
    receipt: ArbReceipt,
    tx: &ArbTransactionSigned,
    gas_used: u64,
) -> alloy_serde::WithOtherFields<alloy_rpc_types_eth::TransactionReceipt> {
    let input = ConvertReceiptInput {
        receipt,
        tx: Recovered::new_unchecked(tx, SIGNER),
        gas_used,
        next_log_index: 0,
        meta: meta(),
    };
    ArbReceiptConverter
        .convert_receipts(vec![input])
        .expect("convert ok")
        .pop()
        .expect("one receipt")
}

#[test]
fn legacy_receipt_has_no_type_override() {
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    let tx = legacy_tx();
    let out = convert_one(r, &tx, 21_000);
    assert!(!out.other.contains_key("type"));
    assert_eq!(
        out.other.get("gasUsedForL1").and_then(|v| v.as_str()),
        Some("0x0")
    );
    assert!(!out.other.contains_key("l1BlockNumber"));
    assert_eq!(out.gas_used, 21_000);
}

#[test]
fn deposit_receipt_overrides_type_to_0x64() {
    let r = ArbReceipt::new(ArbReceiptKind::Deposit(ArbDepositReceipt));
    let tx = deposit_tx();
    let out = convert_one(r, &tx, 0);
    assert_eq!(out.other.get("type").and_then(|v| v.as_str()), Some("0x64"));
}

#[test]
fn retry_receipt_overrides_type_to_0x68() {
    let r = ArbReceipt::new(ArbReceiptKind::Retry(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 50_000,
        logs: vec![],
    }))
    .with_gas_used_for_l1(1234);
    let tx = retry_tx();
    let out = convert_one(r, &tx, 50_000);
    assert_eq!(out.other.get("type").and_then(|v| v.as_str()), Some("0x68"));
    assert_eq!(
        out.other.get("gasUsedForL1").and_then(|v| v.as_str()),
        Some("0x4d2")
    );
}

#[test]
fn submit_retryable_receipt_overrides_type_to_0x69() {
    let r = ArbReceipt::new(ArbReceiptKind::SubmitRetryable(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 200_000,
        logs: vec![],
    }));
    let tx = submit_retryable_tx();
    let out = convert_one(r, &tx, 200_000);
    assert_eq!(out.other.get("type").and_then(|v| v.as_str()), Some("0x69"));
}

#[test]
fn internal_receipt_overrides_type_to_0x6a() {
    let r = ArbReceipt::new(ArbReceiptKind::Internal(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 0,
        logs: vec![],
    }));
    let tx = internal_tx();
    let out = convert_one(r, &tx, 0);
    assert_eq!(out.other.get("type").and_then(|v| v.as_str()), Some("0x6a"));
}

#[test]
fn receipt_includes_multi_gas_used_when_nonzero() {
    let mut r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    r.multi_gas_used = MultiGas::new(arb_primitives::multigas::ResourceKind::Computation, 100);
    let tx = legacy_tx();
    let out = convert_one(r, &tx, 21_000);
    assert!(out.other.contains_key("multiGasUsed"));
}

#[test]
fn receipt_omits_multi_gas_used_when_zero() {
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    let tx = legacy_tx();
    let out = convert_one(r, &tx, 21_000);
    assert!(!out.other.contains_key("multiGasUsed"));
}

#[test]
fn receipt_logs_carry_block_and_tx_metadata() {
    let log = Log {
        address: address!("00000000000000000000000000000000000000cc"),
        data: alloy_primitives::LogData::new_unchecked(
            vec![B256::repeat_byte(0xEE)],
            Bytes::from(vec![0x42; 32]),
        ),
    };
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![log],
    }));
    let tx = legacy_tx();
    let out = convert_one(r, &tx, 21_000);
    let rpc_logs = out.inner.logs();
    assert_eq!(rpc_logs.len(), 1);
    assert_eq!(rpc_logs[0].block_number, Some(42));
    assert_eq!(rpc_logs[0].block_hash, Some(B256::repeat_byte(0xBB)));
    assert_eq!(rpc_logs[0].log_index, Some(0));
}

// ── convert_receipts_with_block paths ─────────────────────────────────────

fn block_with_mix_hash(mix: [u8; 32]) -> SealedBlock<Block<ArbTransactionSigned>> {
    let header = Header {
        mix_hash: B256::from(mix),
        ..Default::default()
    };
    SealedBlock::seal_slow(Block::new(header, BlockBody::default()))
}

#[test]
fn with_block_emits_l1_block_number_from_mix_hash() {
    let mut mix = [0u8; 32];
    mix[8..16].copy_from_slice(&0x1234u64.to_be_bytes());
    let block = block_with_mix_hash(mix);
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    let tx = legacy_tx();
    let out = ArbReceiptConverter
        .convert_receipts_with_block(
            vec![ConvertReceiptInput {
                receipt: r,
                tx: Recovered::new_unchecked(&tx, SIGNER),
                gas_used: 21_000,
                next_log_index: 0,
                meta: meta(),
            }],
            &block,
        )
        .unwrap()
        .pop()
        .unwrap();
    assert_eq!(
        out.other.get("l1BlockNumber").and_then(|v| v.as_str()),
        Some("0x1234")
    );
}

#[test]
fn with_block_legacy_uses_gas_price_when_collect_tips_set() {
    let mut mix = [0u8; 32];
    // mix_hash[16..24] = arbos_version. Pick v60 to use the post-v9 encoding.
    mix[16..24].copy_from_slice(&60u64.to_be_bytes());
    // mix_hash[25] bit 0 = CollectTips
    mix[25] = 1;
    let block = block_with_mix_hash(mix);
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    let tx = legacy_tx();
    let out = ArbReceiptConverter
        .convert_receipts_with_block(
            vec![ConvertReceiptInput {
                receipt: r,
                tx: Recovered::new_unchecked(&tx, SIGNER),
                gas_used: 21_000,
                next_log_index: 0,
                meta: meta(),
            }],
            &block,
        )
        .unwrap()
        .pop()
        .unwrap();
    assert_eq!(out.inner.effective_gas_price, 1_000_000_000u128);
}

#[test]
fn with_block_deposit_effective_gas_price_zero_when_collect_tips() {
    let mut mix = [0u8; 32];
    mix[16..24].copy_from_slice(&60u64.to_be_bytes());
    mix[25] = 1;
    let block = block_with_mix_hash(mix);
    let r = ArbReceipt::new(ArbReceiptKind::Deposit(ArbDepositReceipt));
    let tx = deposit_tx();
    let out = ArbReceiptConverter
        .convert_receipts_with_block(
            vec![ConvertReceiptInput {
                receipt: r,
                tx: Recovered::new_unchecked(&tx, SIGNER),
                gas_used: 0,
                next_log_index: 0,
                meta: meta(),
            }],
            &block,
        )
        .unwrap()
        .pop()
        .unwrap();
    assert_eq!(out.inner.effective_gas_price, 0);
}

#[test]
fn with_block_v9_legacy_encoding_forces_collect_tips() {
    let mut mix = [0u8; 32];
    // ArbOS v9 is the "CollectTipsOld" version — always implies CollectTips=true.
    mix[16..24].copy_from_slice(&9u64.to_be_bytes());
    // bit 0 explicitly zero → still treated as CollectTips.
    let block = block_with_mix_hash(mix);
    let r = ArbReceipt::new(ArbReceiptKind::Legacy(AlloyReceipt {
        status: true.into(),
        cumulative_gas_used: 21_000,
        logs: vec![],
    }));
    let tx = legacy_tx();
    let out = ArbReceiptConverter
        .convert_receipts_with_block(
            vec![ConvertReceiptInput {
                receipt: r,
                tx: Recovered::new_unchecked(&tx, SIGNER),
                gas_used: 21_000,
                next_log_index: 0,
                meta: meta(),
            }],
            &block,
        )
        .unwrap()
        .pop()
        .unwrap();
    assert_eq!(out.inner.effective_gas_price, 1_000_000_000u128);
}
