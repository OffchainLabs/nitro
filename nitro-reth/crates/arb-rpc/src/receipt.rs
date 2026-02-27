//! Arbitrum receipt conversion for RPC responses.

use alloy_consensus::{Receipt, ReceiptEnvelope, ReceiptWithBloom, TxReceipt, Typed2718};
use alloy_primitives::{Address, Bloom, TxKind};
use alloy_rpc_types_eth::TransactionReceipt;
use arb_primitives::ArbPrimitives;
use reth_rpc_convert::transaction::{ConvertReceiptInput, ReceiptConverter};
use reth_rpc_eth_types::EthApiError;

/// Converts Arbitrum receipts to RPC transaction receipts.
#[derive(Debug, Clone)]
pub struct ArbReceiptConverter;

impl ReceiptConverter<ArbPrimitives> for ArbReceiptConverter {
    type RpcReceipt = TransactionReceipt;
    type Error = EthApiError;

    fn convert_receipts(
        &self,
        receipts: Vec<ConvertReceiptInput<'_, ArbPrimitives>>,
    ) -> Result<Vec<TransactionReceipt>, EthApiError> {
        let results = receipts
            .into_iter()
            .map(convert_single_receipt)
            .collect();
        Ok(results)
    }
}

fn convert_single_receipt(input: ConvertReceiptInput<'_, ArbPrimitives>) -> TransactionReceipt {
    use alloy_consensus::{transaction::TxHashRef, Transaction};

    let ConvertReceiptInput {
        receipt,
        tx,
        gas_used,
        next_log_index,
        meta,
    } = input;

    let from = tx.signer();
    let tx_hash = *tx.tx_hash();
    let tx_type = tx.ty();

    let (contract_address, to) = match tx.kind() {
        TxKind::Create => (Some(from.create(tx.nonce())), None),
        TxKind::Call(addr) => (None, Some(Address(*addr))),
    };

    let cumulative_gas_used = receipt.cumulative_gas_used();
    let status = receipt.status_or_post_state();

    // Convert primitive logs to RPC logs with block/tx metadata.
    let rpc_logs: Vec<alloy_rpc_types_eth::Log> = receipt
        .logs()
        .iter()
        .enumerate()
        .map(|(i, log)| alloy_rpc_types_eth::Log {
            inner: log.clone(),
            block_hash: Some(meta.block_hash),
            block_number: Some(meta.block_number),
            block_timestamp: None,
            transaction_hash: Some(tx_hash),
            transaction_index: Some(meta.index),
            log_index: Some(next_log_index as u64 + i as u64),
            removed: false,
        })
        .collect();

    let bloom: Bloom = receipt.logs().iter().collect();

    let receipt_with_bloom = ReceiptWithBloom::new(
        Receipt {
            status,
            cumulative_gas_used,
            logs: rpc_logs,
        },
        bloom,
    );

    // Build envelope matching transaction type.
    let envelope = match tx_type {
        0x01 => ReceiptEnvelope::Eip2930(receipt_with_bloom),
        0x02 => ReceiptEnvelope::Eip1559(receipt_with_bloom),
        0x03 => ReceiptEnvelope::Eip4844(receipt_with_bloom),
        0x04 => ReceiptEnvelope::Eip7702(receipt_with_bloom),
        _ => ReceiptEnvelope::Legacy(receipt_with_bloom),
    };

    // Internal (0x64), deposit (0x6a), and submit retryable (0x69) txs have no gas cost.
    let (effective_gas_used, effective_gas_price) = match tx_type {
        0x64 | 0x69 | 0x6a => (0u64, 0u128),
        _ => (gas_used, meta.base_fee.unwrap_or(0) as u128),
    };

    TransactionReceipt {
        inner: envelope,
        transaction_hash: tx_hash,
        transaction_index: Some(meta.index),
        block_hash: Some(meta.block_hash),
        block_number: Some(meta.block_number),
        gas_used: effective_gas_used,
        effective_gas_price,
        blob_gas_used: None,
        blob_gas_price: None,
        from,
        to,
        contract_address,
    }
}
