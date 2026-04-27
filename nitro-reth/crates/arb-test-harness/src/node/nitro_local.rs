//! Local Nitro subprocess backend (the oracle).
//!
//! Boots Nitro from `NITRO_REF_BINARY` with the no-L1 flags
//! (`--init.empty=true`, `--node.parent-chain-reader.enable=false`,
//! `--node.dangerous.no-l1-listener=true`,
//! `--execution.rpc-server.{enable,public,authenticated}=true`,...) and
//! exposes the same RPC surface as arbreth. Real impl lands via Agent A.

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

pub struct NitroProcess {
    rpc_url: String,
}

impl NitroProcess {
    pub fn start(_ctx: &NodeStartCtx) -> Result<Self> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::start (Stage 2 / Agent A)",
        })
    }
}

impl ExecutionNode for NitroProcess {
    fn kind(&self) -> NodeKind {
        NodeKind::NitroLocal
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
            what: "NitroProcess::submit_message",
        })
    }

    fn block(&self, _id: BlockId) -> Result<Block> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::block",
        })
    }

    fn receipt(&self, _tx: B256) -> Result<TxReceipt> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::receipt",
        })
    }

    fn arb_receipt(&self, _tx: B256) -> Result<ArbReceiptFields> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::arb_receipt",
        })
    }

    fn storage(&self, _addr: Address, _slot: B256, _at: BlockId) -> Result<B256> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::storage",
        })
    }

    fn balance(&self, _addr: Address, _at: BlockId) -> Result<U256> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::balance",
        })
    }

    fn nonce(&self, _addr: Address, _at: BlockId) -> Result<u64> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::nonce",
        })
    }

    fn code(&self, _addr: Address, _at: BlockId) -> Result<Bytes> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::code",
        })
    }

    fn eth_call(&self, _tx: TxRequest, _at: BlockId) -> Result<Bytes> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::eth_call",
        })
    }

    fn debug_storage_range(
        &self,
        _addr: Address,
        _at: BlockId,
    ) -> Result<BTreeMap<B256, B256>> {
        Err(HarnessError::NotImplemented {
            what: "NitroProcess::debug_storage_range",
        })
    }

    fn shutdown(self: Box<Self>) -> Result<()> {
        Ok(())
    }
}
