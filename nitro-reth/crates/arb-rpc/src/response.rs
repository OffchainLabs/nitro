//! Arbitrum transaction response conversion.

use alloy_primitives::Address;
use alloy_rpc_types_eth::{Transaction, TransactionInfo};
use alloy_serde::WithOtherFields;
use arb_primitives::ArbTransactionSigned;
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

        let base =
            Transaction::from_transaction(Recovered::new_unchecked(tx, signer), tx_info);

        Ok(WithOtherFields::new(base))
    }
}
