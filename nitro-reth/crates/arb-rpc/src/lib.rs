//! Arbitrum-specific RPC types, converters, and builders.
//!
//! Provides the Arbitrum Eth API builder and RPC type conversions needed
//! to serve the `eth_` namespace with Arbitrum-specific transaction,
//! receipt, and header types.

pub mod api;
pub mod arb_api;
pub mod arbdebug;
pub mod arbtimeboost;
pub mod arbtrace;
pub mod block_producer;
pub mod builder;
pub mod header;
pub mod nitro_execution;
pub mod nitro_execution_handler;
pub mod nodeinterface_rpc;
pub mod receipt;
pub mod response;
pub mod transaction;
pub mod types;

pub use api::ArbEthApi;
pub use arb_api::{ArbApiHandler, ArbApiServer};
pub use block_producer::{BlockProducer, BlockProducerError, BlockProductionInput, ProducedBlock};
pub use builder::{ArbEthApiBuilder, ArbRpcConvert};
pub use header::ArbHeaderConverter;
pub use nitro_execution::{NitroExecutionApiServer, RpcMessageResult, RpcMessageWithMetadata};
pub use nitro_execution_handler::NitroExecutionHandler;
pub use receipt::ArbReceiptConverter;
pub use response::ArbRpcTxConverter;
pub use transaction::ArbTransactionRequest;
pub use types::ArbRpcTypes;

use alloy_primitives::{B256, U256};
use serde::{Deserialize, Serialize};

/// Arbitrum transaction receipt extension fields.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbReceiptFields {
    /// L1 block number when the L2 tx was batched.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub l1_block_number: Option<u64>,
    /// Gas units charged for L1 calldata.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gas_used_for_l1: Option<U256>,
}

/// Arbitrum block information returned by `arb_getBlockInfo`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbBlockInfo {
    /// The L1 block number that this L2 block was batched on.
    pub l1_block_number: u64,
    /// ArbOS format version of this block.
    pub arbos_format_version: u64,
    /// The send count (L2-to-L1 messages) as of this block.
    pub send_count: u64,
    /// The send root hash.
    pub send_root: B256,
}

/// Maintenance status for the node.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbMaintenanceStatus {
    pub is_running: bool,
}

/// Nitro-parity response for `arb_getRawBlockMetadata`.
///
/// Per block in the queried range, returns the block number and the raw
/// metadata bytes. In arbreth we currently do not maintain a separate
/// metadata sidecar (Nitro's `bulkBlockMetadataFetcher`), so the bytes
/// are empty — consumers should interpret "empty rawMetadata" as "no
/// metadata stored", which matches the Nitro schema.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct NumberAndBlockMetadata {
    pub block_number: u64,
    pub raw_metadata: alloy_primitives::Bytes,
}
