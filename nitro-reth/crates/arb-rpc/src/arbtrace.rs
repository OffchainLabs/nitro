//! `arbtrace_*` RPC namespace — forwards pre-Nitro trace requests to a
//! configured classic (pre-Nitro) node.
//!
//! Arbitrum's `arbtrace_*` methods mirror the Parity/OpenEthereum trace
//! API but are constrained to blocks before the Nitro genesis fork.
//! Nitro itself doesn't implement them against its own chain — it
//! forwards to a classic-node RPC endpoint when configured, or returns
//! an error if no endpoint is set (see `execution/gethexec/api.go:350`).
//!
//! We replicate that behavior exactly: expose the 8 methods on the
//! `arbtrace` namespace, return an "arbtrace calls forwarding not
//! configured" error when no fallback URL is provided, and reject
//! block numbers past the Nitro genesis block.

use jsonrpsee::{
    core::RpcResult,
    proc_macros::rpc,
    types::{error::INTERNAL_ERROR_CODE, ErrorObject},
};
use serde_json::{self as json, value::RawValue, Value as JsonValue};
use std::sync::Arc;

/// Error surface for arbtrace-forwarding operations.
fn forwarding_not_configured() -> ErrorObject<'static> {
    ErrorObject::owned(
        INTERNAL_ERROR_CODE,
        "arbtrace calls forwarding not configured",
        None::<()>,
    )
}

fn block_unsupported_by_classic(block_num: i64, genesis: u64) -> ErrorObject<'static> {
    ErrorObject::owned(
        INTERNAL_ERROR_CODE,
        format!("block number {block_num} is not supported by classic node (> genesis {genesis})"),
        None::<()>,
    )
}

/// Configuration for the `arbtrace` namespace.
#[derive(Debug, Clone, Default)]
pub struct ArbTraceConfig {
    /// URL of the pre-Nitro classic node to forward requests to. When
    /// None, every `arbtrace_*` method returns
    /// "forwarding not configured" per Nitro's canonical behavior.
    pub fallback_client_url: Option<String>,
    /// Nitro genesis block number. Requests targeting blocks above this
    /// are rejected without contacting the classic node.
    pub genesis_block_num: u64,
}

/// `arbtrace` RPC namespace.
#[rpc(server, namespace = "arbtrace")]
pub trait ArbTraceApi {
    /// trace_call equivalent against a historical (pre-Nitro) block.
    #[method(name = "call")]
    async fn call(
        &self,
        call_args: Box<RawValue>,
        trace_types: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    /// trace_callMany equivalent.
    #[method(name = "callMany")]
    async fn call_many(
        &self,
        calls: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    /// trace_replayBlockTransactions equivalent.
    #[method(name = "replayBlockTransactions")]
    async fn replay_block_transactions(
        &self,
        block_num_or_hash: Box<RawValue>,
        trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    /// trace_replayTransaction equivalent.
    #[method(name = "replayTransaction")]
    async fn replay_transaction(
        &self,
        tx_hash: Box<RawValue>,
        trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    /// trace_transaction equivalent.
    #[method(name = "transaction")]
    async fn transaction(&self, tx_hash: Box<RawValue>) -> RpcResult<JsonValue>;

    /// trace_get equivalent — retrieves a sub-trace at a given path.
    #[method(name = "get")]
    async fn get(&self, tx_hash: Box<RawValue>, path: Box<RawValue>) -> RpcResult<JsonValue>;

    /// trace_block equivalent.
    #[method(name = "block")]
    async fn block(&self, block_num_or_hash: Box<RawValue>) -> RpcResult<JsonValue>;

    /// trace_filter equivalent.
    #[method(name = "filter")]
    async fn filter(&self, filter: Box<RawValue>) -> RpcResult<JsonValue>;
}

/// Handler implementing the `arbtrace` namespace. Forwards to the
/// configured classic node when one is set, otherwise returns
/// "forwarding not configured".
#[derive(Debug, Clone)]
pub struct ArbTraceHandler {
    config: Arc<ArbTraceConfig>,
}

impl ArbTraceHandler {
    pub fn new(config: ArbTraceConfig) -> Self {
        Self {
            config: Arc::new(config),
        }
    }

    fn check_block_supported_by_classic(
        &self,
        block_num_or_hash: &RawValue,
    ) -> Result<(), ErrorObject<'static>> {
        // Parse JSON — accept either a hex block number, a tag (latest,
        // earliest, …), or a { "blockHash": "..." } object. Only reject
        // when the value is clearly a number above genesis.
        let parsed: JsonValue = json::from_str(block_num_or_hash.get()).unwrap_or(JsonValue::Null);
        if let Some(s) = parsed.as_str() {
            if let Some(hex) = s.strip_prefix("0x").or_else(|| s.strip_prefix("0X")) {
                if let Ok(n) = i64::from_str_radix(hex, 16) {
                    if n < 0 || (n as u64) > self.config.genesis_block_num {
                        return Err(block_unsupported_by_classic(
                            n,
                            self.config.genesis_block_num,
                        ));
                    }
                }
            }
        }
        Ok(())
    }

    /// All 8 `arbtrace_*` methods share a common tail: "not configured"
    /// since arbreth has no fallback RPC client wired yet. Extracted so
    /// the trait impls stay tight.
    fn forward_not_configured(&self) -> RpcResult<JsonValue> {
        if self.config.fallback_client_url.is_none() {
            return Err(forwarding_not_configured());
        }
        // TODO: when a fallback client is wired, issue a JSON-RPC
        // request via reqwest/jsonrpsee-http-client and return the
        // RawMessage back. For now, we explicitly signal that the
        // classic node integration isn't yet available.
        Err(ErrorObject::owned(
            INTERNAL_ERROR_CODE,
            "arbtrace classic-node forwarding is not yet implemented",
            None::<()>,
        ))
    }
}

#[async_trait::async_trait]
impl ArbTraceApiServer for ArbTraceHandler {
    async fn call(
        &self,
        _call_args: Box<RawValue>,
        _trace_types: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward_not_configured()
    }

    async fn call_many(
        &self,
        _calls: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward_not_configured()
    }

    async fn replay_block_transactions(
        &self,
        block_num_or_hash: Box<RawValue>,
        _trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward_not_configured()
    }

    async fn replay_transaction(
        &self,
        _tx_hash: Box<RawValue>,
        _trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.forward_not_configured()
    }

    async fn transaction(&self, _tx_hash: Box<RawValue>) -> RpcResult<JsonValue> {
        self.forward_not_configured()
    }

    async fn get(&self, _tx_hash: Box<RawValue>, _path: Box<RawValue>) -> RpcResult<JsonValue> {
        self.forward_not_configured()
    }

    async fn block(&self, block_num_or_hash: Box<RawValue>) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward_not_configured()
    }

    async fn filter(&self, _filter: Box<RawValue>) -> RpcResult<JsonValue> {
        self.forward_not_configured()
    }
}
