//! arbreth subprocess backend.
//!
//! Spawns the binary at `ARB_SPEC_BINARY` (or [`NodeStartCtx::binary`])
//! against a freshly-written genesis JSON, then talks to it via
//! standard JSON-RPC. Implementation lands in Stage 2 (Agent A).

use std::collections::BTreeMap;

use alloy_primitives::{Address, Bytes, B256, U256};

use crate::{
    error::HarnessError,
    messaging::L1Message,
    node::{
        ArbReceiptFields, Block, BlockId, ExecutionNode, NodeKind, NodeStartCtx, TxReceipt,
        TxRequest,
    },
    Result,
};

pub struct ArbrethProcess {
    rpc_url: String,
    // child: std::process::Child  ← populated by Agent A
    // workdir: std::path::PathBuf
}

impl ArbrethProcess {
    pub fn start(_ctx: &NodeStartCtx) -> Result<Self> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::start (Stage 2 / Agent A)",
        })
    }
}

impl ExecutionNode for ArbrethProcess {
    fn kind(&self) -> NodeKind {
        NodeKind::Arbreth
    }

    fn rpc_url(&self) -> &str {
        &self.rpc_url
    }

    fn submit_message(
        &mut self,
        _idx: u64,
        _msg: &L1Message,
        _delayed_messages_read: u64,
    ) -> Result<()> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::submit_message",
        })
    }

    fn block(&self, _id: BlockId) -> Result<Block> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::block",
        })
    }

    fn receipt(&self, _tx: B256) -> Result<TxReceipt> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::receipt",
        })
    }

    fn arb_receipt(&self, _tx: B256) -> Result<ArbReceiptFields> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::arb_receipt",
        })
    }

    fn storage(&self, _addr: Address, _slot: B256, _at: BlockId) -> Result<B256> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::storage",
        })
    }

    fn balance(&self, _addr: Address, _at: BlockId) -> Result<U256> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::balance",
        })
    }

    fn nonce(&self, _addr: Address, _at: BlockId) -> Result<u64> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::nonce",
        })
    }

    fn code(&self, _addr: Address, _at: BlockId) -> Result<Bytes> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::code",
        })
    }

    fn eth_call(&self, _tx: TxRequest, _at: BlockId) -> Result<Bytes> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::eth_call",
        })
    }

    fn debug_storage_range(
        &self,
        _addr: Address,
        _at: BlockId,
    ) -> Result<BTreeMap<B256, B256>> {
        Err(HarnessError::NotImplemented {
            what: "ArbrethProcess::debug_storage_range",
        })
    }

    fn shutdown(self: Box<Self>) -> Result<()> {
        Ok(())
    }
}
