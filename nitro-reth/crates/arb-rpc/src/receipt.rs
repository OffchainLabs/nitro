//! Arbitrum receipt conversion for RPC responses.

use alloy_consensus::{Receipt, ReceiptEnvelope, ReceiptWithBloom, TxReceipt, Typed2718};
use alloy_primitives::{Address, Bloom, TxKind};
use alloy_rpc_types_eth::TransactionReceipt;
use alloy_serde::WithOtherFields;
use arb_primitives::ArbPrimitives;
use reth_primitives_traits::SealedBlock;
use reth_rpc_convert::transaction::{ConvertReceiptInput, ReceiptConverter};
use reth_rpc_eth_types::EthApiError;

use crate::header::l1_block_number_from_mix_hash;

/// Converts Arbitrum receipts to RPC transaction receipts with extension fields.
#[derive(Debug, Clone)]
pub struct ArbReceiptConverter;

impl ReceiptConverter<ArbPrimitives> for ArbReceiptConverter {
    type RpcReceipt = WithOtherFields<TransactionReceipt>;
    type Error = EthApiError;

    fn convert_receipts(
        &self,
        receipts: Vec<ConvertReceiptInput<'_, ArbPrimitives>>,
    ) -> Result<Vec<Self::RpcReceipt>, EthApiError> {
        let results = receipts
            .into_iter()
            .map(|input| convert_single_receipt(input, None))
            .collect();
        Ok(results)
    }

    fn convert_receipts_with_block(
        &self,
        receipts: Vec<ConvertReceiptInput<'_, ArbPrimitives>>,
        block: &SealedBlock<alloy_consensus::Block<arb_primitives::ArbTransactionSigned>>,
    ) -> Result<Vec<Self::RpcReceipt>, Self::Error> {
        let mix_hash = block.header().mix_hash;
        let l1_block_number = l1_block_number_from_mix_hash(&mix_hash);

        let results = receipts
            .into_iter()
            .map(|input| convert_single_receipt(input, Some(l1_block_number)))
            .collect();
        Ok(results)
    }
}

fn convert_single_receipt(
    input: ConvertReceiptInput<'_, ArbPrimitives>,
    l1_block_number: Option<u64>,
) -> WithOtherFields<TransactionReceipt> {
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
    let gas_used_for_l1 = receipt.gas_used_for_l1;

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

    // On Arbitrum, effective gas price is always the block base fee for all tx types.
    let effective_gas_price = meta.base_fee.unwrap_or(0) as u128;

    let base_receipt = TransactionReceipt {
        inner: envelope,
        transaction_hash: tx_hash,
        transaction_index: Some(meta.index),
        block_hash: Some(meta.block_hash),
        block_number: Some(meta.block_number),
        gas_used,
        effective_gas_price,
        blob_gas_used: None,
        blob_gas_price: None,
        from,
        to,
        contract_address,
    };

    // Add Arbitrum-specific extension fields.
    let mut other = std::collections::BTreeMap::new();

    // Override `type` for Arbitrum tx types (0x64+) since ReceiptEnvelope
    // only supports standard Ethereum types and falls back to Legacy (0x0).
    if tx_type >= 0x64 {
        other.insert(
            "type".to_string(),
            serde_json::to_value(format!("{tx_type:#x}")).unwrap_or_default(),
        );
    }

    // gasUsedForL1: always present on Arbitrum receipts.
    other.insert(
        "gasUsedForL1".to_string(),
        serde_json::to_value(format!("{:#x}", gas_used_for_l1)).unwrap_or_default(),
    );

    // l1BlockNumber: included when block header is available.
    if let Some(l1_bn) = l1_block_number {
        other.insert(
            "l1BlockNumber".to_string(),
            serde_json::to_value(format!("{l1_bn:#x}")).unwrap_or_default(),
        );
    }

    // multiGasUsed: multi-dimensional gas breakdown.
    if !receipt.multi_gas_used.is_zero() {
        other.insert(
            "multiGasUsed".to_string(),
            serde_json::to_value(&receipt.multi_gas_used).unwrap_or_default(),
        );
    }

    WithOtherFields {
        inner: base_receipt,
        other: alloy_serde::OtherFields::new(other),
    }
}
