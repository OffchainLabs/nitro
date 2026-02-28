//! Arbitrum transaction response conversion.

use alloy_primitives::Address;
use alloy_rpc_types_eth::{Transaction, TransactionInfo};
use alloy_serde::WithOtherFields;
use arb_primitives::{ArbTransactionSigned, ArbTypedTransaction};
use reth_rpc_convert::transaction::RpcTxConverter;
use std::convert::Infallible;

/// Converts consensus transactions to RPC transaction responses.
#[derive(Debug, Clone)]
pub struct ArbRpcTxConverter;

impl RpcTxConverter<ArbTransactionSigned, WithOtherFields<Transaction<ArbTransactionSigned>>, TransactionInfo>
    for ArbRpcTxConverter
{
    type Err = Infallible;

    fn convert_rpc_tx(
        &self,
        tx: ArbTransactionSigned,
        signer: Address,
        tx_info: TransactionInfo,
    ) -> Result<WithOtherFields<Transaction<ArbTransactionSigned>>, Infallible> {
        use alloy_consensus::transaction::Recovered;

        let other = arb_tx_fields(tx.inner());

        let base =
            Transaction::from_transaction(Recovered::new_unchecked(tx, signer), tx_info);

        Ok(WithOtherFields {
            inner: base,
            other: alloy_serde::OtherFields::new(other),
        })
    }
}

/// Extract Arbitrum-specific extension fields from a transaction.
fn arb_tx_fields(tx: &ArbTypedTransaction) -> std::collections::BTreeMap<String, serde_json::Value> {
    let mut fields = std::collections::BTreeMap::new();

    match tx {
        ArbTypedTransaction::Deposit(d) => {
            fields.insert(
                "requestId".to_string(),
                serde_json::to_value(d.l1_request_id).unwrap_or_default(),
            );
        }
        ArbTypedTransaction::Contract(c) => {
            fields.insert(
                "requestId".to_string(),
                serde_json::to_value(c.request_id).unwrap_or_default(),
            );
        }
        ArbTypedTransaction::Retry(r) => {
            fields.insert(
                "ticketId".to_string(),
                serde_json::to_value(r.ticket_id).unwrap_or_default(),
            );
            fields.insert(
                "refundTo".to_string(),
                serde_json::to_value(r.refund_to).unwrap_or_default(),
            );
            fields.insert(
                "maxRefund".to_string(),
                serde_json::to_value(r.max_refund).unwrap_or_default(),
            );
            fields.insert(
                "submissionFeeRefund".to_string(),
                serde_json::to_value(r.submission_fee_refund).unwrap_or_default(),
            );
        }
        ArbTypedTransaction::SubmitRetryable(s) => {
            fields.insert(
                "requestId".to_string(),
                serde_json::to_value(s.request_id).unwrap_or_default(),
            );
            fields.insert(
                "l1BaseFee".to_string(),
                serde_json::to_value(s.l1_base_fee).unwrap_or_default(),
            );
            fields.insert(
                "depositValue".to_string(),
                serde_json::to_value(s.deposit_value).unwrap_or_default(),
            );
            if let Some(retry_to) = s.retry_to {
                fields.insert(
                    "retryTo".to_string(),
                    serde_json::to_value(retry_to).unwrap_or_default(),
                );
            }
            fields.insert(
                "retryValue".to_string(),
                serde_json::to_value(s.retry_value).unwrap_or_default(),
            );
            fields.insert(
                "beneficiary".to_string(),
                serde_json::to_value(s.beneficiary).unwrap_or_default(),
            );
            fields.insert(
                "maxSubmissionFee".to_string(),
                serde_json::to_value(s.max_submission_fee).unwrap_or_default(),
            );
            fields.insert(
                "refundTo".to_string(),
                serde_json::to_value(s.fee_refund_addr).unwrap_or_default(),
            );
            fields.insert(
                "retryData".to_string(),
                serde_json::to_value(s.retry_data.clone()).unwrap_or_default(),
            );
        }
        // Standard Ethereum types and internal/unsigned: no extra fields.
        _ => {}
    }

    fields
}
