use alloy_evm::Evm;
use alloy_evm::eth::receipt_builder::{ReceiptBuilder, ReceiptBuilderCtx};
use alloy_primitives::Log;

use arb_primitives::{ArbReceipt, ArbTransactionSigned};
use arb_primitives::signed_tx::ArbTxTypeLocal;

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
        let ReceiptBuilderCtx { tx_type, result, cumulative_gas_used, .. } = ctx;
        let success = result.is_success();
        let logs: Vec<Log> = result.into_logs();

        let inner = alloy_consensus::Receipt {
            status: alloy_consensus::Eip658Value::Eip658(success),
            cumulative_gas_used,
            logs,
        };

        match tx_type {
            ArbTxTypeLocal::Legacy => ArbReceipt::Legacy(inner),
            ArbTxTypeLocal::Eip2930 => ArbReceipt::Eip2930(inner),
            ArbTxTypeLocal::Eip1559 => ArbReceipt::Eip1559(inner),
            ArbTxTypeLocal::Eip4844 => ArbReceipt::Eip1559(inner),
            ArbTxTypeLocal::Eip7702 => ArbReceipt::Eip7702(inner),
            ArbTxTypeLocal::Deposit => ArbReceipt::Deposit(arb_primitives::ArbDepositReceipt),
            ArbTxTypeLocal::Unsigned => ArbReceipt::Unsigned(inner),
            ArbTxTypeLocal::Contract => ArbReceipt::Contract(inner),
            ArbTxTypeLocal::Retry => ArbReceipt::Retry(inner),
            ArbTxTypeLocal::SubmitRetryable => ArbReceipt::SubmitRetryable(inner),
            ArbTxTypeLocal::Internal => ArbReceipt::Internal(inner),
        }
    }
}
