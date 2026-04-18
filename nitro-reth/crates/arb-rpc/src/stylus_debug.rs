//! `debug_traceTransaction` override that adds the `stylusTracer`
//! named tracer.
//!
//! When `opts.tracer == "stylusTracer"` the cached Stylus host-I/O
//! records for `tx_hash` are returned as a JSON array; every other
//! tracer name forwards to the standard handler unchanged.

use alloy_primitives::B256;
use alloy_rpc_types_trace::geth::{GethDebugTracerType, GethDebugTracingOptions, GethTrace};
use jsonrpsee::{core::RpcResult, proc_macros::rpc};

use crate::stylus_tracer::take_cached_trace;

pub const STYLUS_TRACER_NAME: &str = "stylusTracer";

/// `debug_*` namespace override exposing the `stylusTracer` option.
#[rpc(server, namespace = "debug")]
pub trait StylusDebug {
    #[method(name = "traceTransaction")]
    async fn trace_transaction(
        &self,
        tx_hash: B256,
        opts: Option<GethDebugTracingOptions>,
    ) -> RpcResult<GethTrace>;
}

/// Async forwarder for non-stylus tracer requests.
pub type DebugForwarder = std::sync::Arc<
    dyn Fn(
            B256,
            Option<GethDebugTracingOptions>,
        )
            -> std::pin::Pin<Box<dyn std::future::Future<Output = RpcResult<GethTrace>> + Send>>
        + Send
        + Sync,
>;

pub struct StylusDebugHandler {
    forwarder: DebugForwarder,
}

impl std::fmt::Debug for StylusDebugHandler {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("StylusDebugHandler").finish()
    }
}

impl StylusDebugHandler {
    pub fn new(forwarder: DebugForwarder) -> Self {
        Self { forwarder }
    }
}

#[async_trait::async_trait]
impl StylusDebugServer for StylusDebugHandler {
    async fn trace_transaction(
        &self,
        tx_hash: B256,
        opts: Option<GethDebugTracingOptions>,
    ) -> RpcResult<GethTrace> {
        if let Some(ref o) = opts {
            if let Some(GethDebugTracerType::JsTracer(name)) = &o.tracer {
                if name == STYLUS_TRACER_NAME {
                    let records = take_cached_trace(tx_hash);
                    let value = serde_json::to_value(&records).unwrap_or_default();
                    return Ok(GethTrace::JS(value));
                }
            }
        }
        (self.forwarder)(tx_hash, opts).await
    }
}
