use std::time::{Duration, Instant};

use crate::{error::HarnessError, Result};

fn build_agent(timeout: Duration) -> ureq::Agent {
    ureq::AgentBuilder::new()
        .timeout(timeout)
        .max_idle_connections(64)
        .max_idle_connections_per_host(32)
        .build()
}

#[derive(Debug, Clone)]
pub struct JsonRpcClient {
    pub url: String,
    pub timeout: Duration,
    agent: ureq::Agent,
}

impl JsonRpcClient {
    pub fn new(url: impl Into<String>) -> Self {
        let timeout = Duration::from_secs(30);
        Self {
            url: url.into(),
            timeout,
            agent: build_agent(timeout),
        }
    }

    pub fn with_timeout(mut self, timeout: Duration) -> Self {
        self.timeout = timeout;
        self.agent = build_agent(timeout);
        self
    }

    pub fn call(&self, method: &str, params: serde_json::Value) -> Result<serde_json::Value> {
        let agent = &self.agent;
        let body = serde_json::json!({
            "jsonrpc": "2.0",
            "id": 1,
            "method": method,
            "params": params,
        });
        let resp = agent
            .post(&self.url)
            .set("Content-Type", "application/json")
            .send_string(&serde_json::to_string(&body)?)
            .map_err(|e| HarnessError::Rpc(format!("{method}: transport: {e}")))?;
        let json: serde_json::Value = resp
            .into_json()
            .map_err(|e| HarnessError::Rpc(format!("{method}: decode: {e}")))?;
        if let Some(err) = json.get("error") {
            return Err(HarnessError::Rpc(format!("{method}: {err}")));
        }
        Ok(json
            .get("result")
            .cloned()
            .unwrap_or(serde_json::Value::Null))
    }

    /// Retries on transport errors only; JSON-RPC errors propagate immediately.
    #[allow(unused)]
    pub fn call_with_retry(
        &self,
        method: &str,
        params: serde_json::Value,
        deadline: Instant,
    ) -> Result<serde_json::Value> {
        loop {
            match self.call(method, params.clone()) {
                Ok(v) => return Ok(v),
                Err(e) => {
                    let is_transport =
                        matches!(&e, HarnessError::Rpc(m) if m.contains("transport"));
                    if !is_transport {
                        return Err(e);
                    }
                    if Instant::now() >= deadline {
                        return Err(e);
                    }
                    std::thread::sleep(Duration::from_millis(200));
                }
            }
        }
    }
}
