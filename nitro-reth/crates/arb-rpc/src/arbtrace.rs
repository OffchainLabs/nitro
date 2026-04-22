//! `arbtrace_*` namespace — forwards pre-Nitro trace requests to a
//! configured classic-node RPC endpoint.

use std::sync::Arc;

use jsonrpsee::{
    core::{client::ClientT, RpcResult},
    proc_macros::rpc,
    types::{error::INTERNAL_ERROR_CODE, ErrorObject},
};
use jsonrpsee_http_client::{HttpClient, HttpClientBuilder};
use parking_lot::Mutex;
use serde_json::{self as json, value::RawValue, Value as JsonValue};
use std::time::Duration;

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

fn http_error(e: impl std::fmt::Display) -> ErrorObject<'static> {
    ErrorObject::owned(
        INTERNAL_ERROR_CODE,
        format!("arbtrace forwarding failed: {e}"),
        None::<()>,
    )
}

#[derive(Debug, Clone, Default)]
pub struct ArbTraceConfig {
    /// URL of the pre-Nitro classic node to forward requests to.
    pub fallback_client_url: Option<String>,
    /// Timeout for forwarded RPC calls.
    pub fallback_client_timeout: Option<Duration>,
    /// Nitro genesis block number; requests past this are rejected
    /// without hitting the classic node.
    pub genesis_block_num: u64,
}

#[rpc(server, namespace = "arbtrace")]
pub trait ArbTraceApi {
    #[method(name = "call")]
    async fn call(
        &self,
        call_args: Box<RawValue>,
        trace_types: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    #[method(name = "callMany")]
    async fn call_many(
        &self,
        calls: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    #[method(name = "replayBlockTransactions")]
    async fn replay_block_transactions(
        &self,
        block_num_or_hash: Box<RawValue>,
        trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    #[method(name = "replayTransaction")]
    async fn replay_transaction(
        &self,
        tx_hash: Box<RawValue>,
        trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue>;

    #[method(name = "transaction")]
    async fn transaction(&self, tx_hash: Box<RawValue>) -> RpcResult<JsonValue>;

    #[method(name = "get")]
    async fn get(&self, tx_hash: Box<RawValue>, path: Box<RawValue>) -> RpcResult<JsonValue>;

    #[method(name = "block")]
    async fn block(&self, block_num_or_hash: Box<RawValue>) -> RpcResult<JsonValue>;

    #[method(name = "filter")]
    async fn filter(&self, filter: Box<RawValue>) -> RpcResult<JsonValue>;
}

/// Lazy HTTP client backed by `jsonrpsee-http-client`.
pub struct ArbTraceHandler {
    config: Arc<ArbTraceConfig>,
    client: Mutex<Option<Arc<HttpClient>>>,
}

impl std::fmt::Debug for ArbTraceHandler {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("ArbTraceHandler")
            .field("config", &self.config)
            .finish_non_exhaustive()
    }
}

impl Clone for ArbTraceHandler {
    fn clone(&self) -> Self {
        Self {
            config: self.config.clone(),
            client: Mutex::new(self.client.lock().clone()),
        }
    }
}

impl ArbTraceHandler {
    pub fn new(config: ArbTraceConfig) -> Self {
        Self {
            config: Arc::new(config),
            client: Mutex::new(None),
        }
    }

    fn get_client(&self) -> Result<Arc<HttpClient>, ErrorObject<'static>> {
        if let Some(c) = self.client.lock().as_ref() {
            return Ok(c.clone());
        }
        let url = self
            .config
            .fallback_client_url
            .as_ref()
            .ok_or_else(forwarding_not_configured)?;
        let mut builder = HttpClientBuilder::default();
        if let Some(t) = self.config.fallback_client_timeout {
            builder = builder.request_timeout(t);
        }
        let client = builder.build(url).map_err(http_error)?;
        let arc = Arc::new(client);
        *self.client.lock() = Some(arc.clone());
        Ok(arc)
    }

    fn check_block_supported_by_classic(
        &self,
        block_num_or_hash: &RawValue,
    ) -> Result<(), ErrorObject<'static>> {
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

    async fn forward(&self, method: &str, params: Vec<Box<RawValue>>) -> RpcResult<JsonValue> {
        let client = self.get_client()?;
        let mut array_params = jsonrpsee::core::params::ArrayParams::new();
        for raw in params {
            let v: JsonValue = serde_json::from_str(raw.get()).unwrap_or(JsonValue::Null);
            array_params.insert(v).map_err(http_error)?;
        }
        let resp: JsonValue = client
            .request(method, array_params)
            .await
            .map_err(http_error)?;
        Ok(resp)
    }
}

#[async_trait::async_trait]
impl ArbTraceApiServer for ArbTraceHandler {
    async fn call(
        &self,
        call_args: Box<RawValue>,
        trace_types: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward(
            "arbtrace_call",
            vec![call_args, trace_types, block_num_or_hash],
        )
        .await
    }

    async fn call_many(
        &self,
        calls: Box<RawValue>,
        block_num_or_hash: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward("arbtrace_callMany", vec![calls, block_num_or_hash])
            .await
    }

    async fn replay_block_transactions(
        &self,
        block_num_or_hash: Box<RawValue>,
        trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward(
            "arbtrace_replayBlockTransactions",
            vec![block_num_or_hash, trace_types],
        )
        .await
    }

    async fn replay_transaction(
        &self,
        tx_hash: Box<RawValue>,
        trace_types: Box<RawValue>,
    ) -> RpcResult<JsonValue> {
        self.forward("arbtrace_replayTransaction", vec![tx_hash, trace_types])
            .await
    }

    async fn transaction(&self, tx_hash: Box<RawValue>) -> RpcResult<JsonValue> {
        self.forward("arbtrace_transaction", vec![tx_hash]).await
    }

    async fn get(&self, tx_hash: Box<RawValue>, path: Box<RawValue>) -> RpcResult<JsonValue> {
        self.forward("arbtrace_get", vec![tx_hash, path]).await
    }

    async fn block(&self, block_num_or_hash: Box<RawValue>) -> RpcResult<JsonValue> {
        self.check_block_supported_by_classic(&block_num_or_hash)?;
        self.forward("arbtrace_block", vec![block_num_or_hash])
            .await
    }

    async fn filter(&self, filter: Box<RawValue>) -> RpcResult<JsonValue> {
        self.forward("arbtrace_filter", vec![filter]).await
    }
}
