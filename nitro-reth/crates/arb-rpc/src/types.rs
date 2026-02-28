//! Arbitrum RPC type definitions.

use alloy_consensus::Header;
use alloy_rpc_types_eth::{Header as RpcHeader, TransactionReceipt};
use alloy_serde::WithOtherFields;
use arb_primitives::ArbTransactionSigned;

use crate::transaction::ArbTransactionRequest;

/// Arbitrum RPC network types.
#[derive(Clone, Debug)]
pub struct ArbRpcTypes;

impl reth_rpc_convert::RpcTypes for ArbRpcTypes {
    type Header = WithOtherFields<RpcHeader<Header>>;
    type Receipt = WithOtherFields<TransactionReceipt>;
    type TransactionResponse =
        WithOtherFields<alloy_rpc_types_eth::Transaction<ArbTransactionSigned>>;
    type TransactionRequest = ArbTransactionRequest;
}
