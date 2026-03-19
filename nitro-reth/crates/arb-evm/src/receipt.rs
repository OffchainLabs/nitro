use alloy_evm::{
    eth::receipt_builder::{ReceiptBuilder, ReceiptBuilderCtx},
    Evm,
};
use alloy_primitives::Log;

use arb_primitives::{signed_tx::ArbTxTypeLocal, ArbReceipt, ArbReceiptKind, ArbTransactionSigned};

/// Builds `ArbReceipt` from execution results.
#[derive(Debug, Clone, Copy, Default)]
pub struct ArbReceiptBuilder;

impl ReceiptBuilder for ArbReceiptBuilder {
    type Transaction = ArbTransactionSigned;
    type Receipt = ArbReceipt;

    fn build_receipt<E: Evm>(
        &self,
        ctx: ReceiptBuilderCtx<'_, ArbTxTypeLocal, E>,
    ) -> Self::Receipt {
        let ReceiptBuilderCtx {
            tx_type,
            result,
            cumulative_gas_used,
            ..
        } = ctx;
        let success = result.is_success();
        let logs: Vec<Log> = result.into_logs();

        let inner = alloy_consensus::Receipt {
            status: alloy_consensus::Eip658Value::Eip658(success),
            cumulative_gas_used,
            logs,
        };

        let kind = match tx_type {
            ArbTxTypeLocal::Legacy => ArbReceiptKind::Legacy(inner),
            ArbTxTypeLocal::Eip2930 => ArbReceiptKind::Eip2930(inner),
            ArbTxTypeLocal::Eip1559 => ArbReceiptKind::Eip1559(inner),
            ArbTxTypeLocal::Eip4844 => ArbReceiptKind::Eip1559(inner),
            ArbTxTypeLocal::Eip7702 => ArbReceiptKind::Eip7702(inner),
            ArbTxTypeLocal::Deposit => ArbReceiptKind::Deposit(arb_primitives::ArbDepositReceipt),
            ArbTxTypeLocal::Unsigned => ArbReceiptKind::Unsigned(inner),
            ArbTxTypeLocal::Contract => ArbReceiptKind::Contract(inner),
            ArbTxTypeLocal::Retry => ArbReceiptKind::Retry(inner),
            ArbTxTypeLocal::SubmitRetryable => ArbReceiptKind::SubmitRetryable(inner),
            ArbTxTypeLocal::Internal => ArbReceiptKind::Internal(inner),
        };

        // gas_used_for_l1 is populated later by the block executor.
        ArbReceipt::new(kind)
    }
}
