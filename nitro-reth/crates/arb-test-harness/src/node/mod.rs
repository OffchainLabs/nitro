//! Abstraction over an Arbitrum execution node (arbreth or Nitro reference).
//!
//! Every implementation exposes the same RPC surface used by the spec-test
//! runner, the differential fuzzer, and the operator CLI. State-altering
//! interactions go through [`ExecutionNode::submit_message`]; everything
//! else is a read.

use std::collections::BTreeMap;

use alloy_primitives::{Address, Bytes, B256, U256};
use serde::{Deserialize, Serialize};

use crate::{messaging::L1Message, Result};

pub mod arbreth;
pub mod nitro_local;
#[cfg(feature = "docker")]
pub mod nitro_docker;

/// Block identifier used in RPC reads. Matches Ethereum's block-tag set
/// plus a numeric form. Always serialized as a string.
#[derive(Debug, Clone)]
pub enum BlockId {
    Number(u64),
    Latest,
    Pending,
    Earliest,
    Finalized,
    Safe,
}

impl BlockId {
    pub fn to_rpc(&self) -> String {
        match self {
            BlockId::Number(n) => format!("0x{n:x}"),
            BlockId::Latest => "latest".into(),
            BlockId::Pending => "pending".into(),
            BlockId::Earliest => "earliest".into(),
            BlockId::Finalized => "finalized".into(),
            BlockId::Safe => "safe".into(),
        }
    }
}

/// Which implementation we're talking to. Used for diagnostics and for
/// behavior that's deliberately permitted to differ (e.g. genesis
/// state-root divergence is expected between Nitro's geth fork and reth).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum NodeKind {
    Arbreth,
    NitroLocal,
    NitroDocker,
}

/// Inputs needed to start a node. Implementations decide what to do
/// with them (subprocess flags, container env, etc.).
#[derive(Debug, Clone)]
pub struct NodeStartCtx {
    /// Path to the binary. Only used for subprocess backends.
    pub binary: Option<String>,
    /// L2 chain id (must agree on both sides of a dual-exec).
    pub l2_chain_id: u64,
    /// L1 chain id served by [`crate::mock_l1::MockL1`].
    pub l1_chain_id: u64,
    /// Mock L1 RPC URL.
    pub mock_l1_rpc: String,
    /// Genesis chain spec JSON (already produced by
    /// [`crate::genesis::GenesisBuilder`]).
    pub genesis: serde_json::Value,
    /// JWT secret for authenticated RPC.
    pub jwt_hex: String,
    /// Working directory for the node (datadir, ipc, logs).
    pub workdir: std::path::PathBuf,
    /// Bind addresses; impls assign free ports if zero.
    pub http_port: u16,
    pub authrpc_port: u16,
}

/// Per-tx Arbitrum-specific receipt fields not present on
/// [`alloy_primitives`]'s standard receipt.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ArbReceiptFields {
    #[serde(default)]
    pub gas_used_for_l1: Option<u64>,
    #[serde(default)]
    pub l1_block_number: Option<u64>,
    #[serde(default)]
    pub multi_gas: Option<MultiGasDims>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MultiGasDims {
    pub computation: u64,
    pub history: u64,
    pub storage: u64,
    pub state_growth: u64,
}

/// Generic block view used by readers. Concrete impls populate from RPC.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Block {
    pub number: u64,
    pub hash: B256,
    pub parent_hash: B256,
    pub state_root: B256,
    pub receipts_root: B256,
    pub transactions_root: B256,
    pub gas_used: u64,
    pub gas_limit: u64,
    pub timestamp: u64,
}

/// Subset of standard tx receipt fields exposed via the trait. Logs
/// captured separately because they are commonly diffed independently.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TxReceipt {
    pub tx_hash: B256,
    pub block_number: u64,
    pub status: u8,
    pub gas_used: u64,
    pub cumulative_gas_used: u64,
    pub effective_gas_price: u128,
    pub from: Address,
    pub to: Option<Address>,
    pub contract_address: Option<Address>,
    pub logs: Vec<EvmLog>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct EvmLog {
    pub address: Address,
    pub topics: Vec<B256>,
    pub data: Bytes,
    pub log_index: u64,
    pub block_number: u64,
    pub tx_hash: B256,
}

/// Minimal eth_call request shape.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TxRequest {
    pub to: Option<Address>,
    pub from: Option<Address>,
    pub data: Option<Bytes>,
    pub value: Option<U256>,
    pub gas: Option<u64>,
}

/// Common interface implemented by every node backend. All methods are
/// blocking (sync). Backends own their RPC client, process handle, and
/// any spawned threads.
pub trait ExecutionNode: Send {
    fn kind(&self) -> NodeKind;

    fn rpc_url(&self) -> &str;

    /// Submit an L1 message via `nitroexecution_digestMessage`.
    fn submit_message(
        &mut self,
        idx: u64,
        msg: &L1Message,
        delayed_messages_read: u64,
    ) -> Result<()>;

    fn block(&self, id: BlockId) -> Result<Block>;

    fn receipt(&self, tx: B256) -> Result<TxReceipt>;

    fn arb_receipt(&self, tx: B256) -> Result<ArbReceiptFields>;

    fn storage(&self, addr: Address, slot: B256, at: BlockId) -> Result<B256>;

    fn balance(&self, addr: Address, at: BlockId) -> Result<U256>;

    fn nonce(&self, addr: Address, at: BlockId) -> Result<u64>;

    fn code(&self, addr: Address, at: BlockId) -> Result<Bytes>;

    fn eth_call(&self, tx: TxRequest, at: BlockId) -> Result<Bytes>;

    fn debug_storage_range(
        &self,
        addr: Address,
        at: BlockId,
    ) -> Result<BTreeMap<B256, B256>>;

    /// Graceful shutdown. Implementations are also expected to clean up
    /// in `Drop`, but explicit shutdown gives the test driver a chance
    /// to surface errors.
    fn shutdown(self: Box<Self>) -> Result<()>;
}
