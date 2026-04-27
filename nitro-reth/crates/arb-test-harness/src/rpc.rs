//! Thin synchronous JSON-RPC client used by the subprocess backends.

use crate::{error::HarnessError, Result};

#[derive(Debug, Clone)]
pub struct JsonRpcClient {
    pub url: String,
    pub timeout: std::time::Duration,
}

impl JsonRpcClient {
    pub fn new(url: impl Into<String>) -> Self {
        Self {
            url: url.into(),
            timeout: std::time::Duration::from_secs(30),
        }
    }

    pub fn call(&self, method: &str, params: serde_json::Value) -> Result<serde_json::Value> {
        let agent = ureq::AgentBuilder::new().timeout(self.timeout).build();
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
}
