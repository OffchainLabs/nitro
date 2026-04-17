//! `debug_traceTransaction` override that adds Nitro's `stylusTracer`
//! named tracer.
//!
//! Nitro registers a `"stylusTracer"` named tracer with go-ethereum's
//! tracer plugin system. Clients call:
//!
//! ```text
//! debug_traceTransaction(tx_hash, {"tracer": "stylusTracer"})
//! ```
//!
//! and receive the captured Stylus host-I/O records inline. We
//! short-circuit that case to the same cache `arb_traceStylusHostio`
//! drains; every other tracer name is forwarded to the standard reth
//! debug handler so behavior is otherwise unchanged.
//!
//! Wiring lives in `arb-node` — it constructs the inner reth `DebugApi`
//! from the registry, wraps it in [`StylusDebugHandler`], and calls
//! `add_or_replace_configured` so the override replaces the stock
//! `debug_traceTransaction` across all configured transports.

use alloy_primitives::B256;
use alloy_rpc_types_trace::geth::{GethDebugTracerType, GethDebugTracingOptions, GethTrace};
use jsonrpsee::{core::RpcResult, proc_macros::rpc};

use crate::stylus_tracer::take_cached_trace;

/// `"stylusTracer"` — Nitro's named tracer constant.
pub const STYLUS_TRACER_NAME: &str = "stylusTracer";

/// `debug_*` namespace override exposing the `stylusTracer` option.
#[rpc(server, namespace = "debug")]
pub trait StylusDebug {
    /// `debug_traceTransaction` with `stylusTracer` support.
    ///
    /// When `opts.tracer == "stylusTracer"`, returns the cached host-I/O
    /// records for `tx_hash` as a JSON array (matching Nitro's wire
    /// shape). Otherwise delegates to the standard reth handler.
    #[method(name = "traceTransaction")]
    async fn trace_transaction(
        &self,
        tx_hash: B256,
        opts: Option<GethDebugTracingOptions>,
    ) -> RpcResult<GethTrace>;
}

/// Async forwarder used to delegate non-stylus tracer requests to the
/// standard reth debug handler. The wiring layer in `arb-node` boxes a
/// closure capturing the live `DebugApi<EthApi>` instance.
pub type DebugForwarder = std::sync::Arc<
    dyn Fn(
            B256,
            Option<GethDebugTracingOptions>,
        )
            -> std::pin::Pin<Box<dyn std::future::Future<Output = RpcResult<GethTrace>> + Send>>
        + Send
        + Sync,
>;

/// Concrete handler. Stores the forwarder; intercepts only the
/// `stylusTracer` case.
pub struct StylusDebugHandler {
    forwarder: DebugForwarder,
}

impl std::fmt::Debug for StylusDebugHandler {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("StylusDebugHandler").finish()
    }
}

impl StylusDebugHandler {
    /// Build a handler from a forwarder closure.
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
