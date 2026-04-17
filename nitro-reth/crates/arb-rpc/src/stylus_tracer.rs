//! Stylus host-I/O tracer: records each WASM host function call made
//! during Stylus program execution so `debug_traceTransaction` can
//! surface them alongside EVM events.

use std::sync::{Arc, Mutex};

use alloy_primitives::{Address, Bytes};
use serde::{Deserialize, Serialize};

/// One host-I/O record captured during Stylus execution.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct HostioTraceInfo {
    /// Host function name (e.g., `storage_load_bytes32`, `contract_call`).
    pub name: String,
    /// Arguments passed to the host function.
    pub args: Bytes,
    /// Outputs returned from the host function.
    pub outs: Bytes,
    /// Ink (gas) counter at entry.
    pub start_ink: u64,
    /// Ink counter at exit.
    pub end_ink: u64,
    /// Target address for CALL/CREATE family host functions.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub address: Option<Address>,
    /// Nested host-I/O records for sub-call frames.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub steps: Vec<HostioTraceInfo>,
}

/// Shared recording buffer — Stylus runtime pushes; debug handler drains.
#[derive(Debug, Default, Clone)]
pub struct StylusTraceBuffer {
    inner: Arc<Mutex<Vec<HostioTraceInfo>>>,
}

impl StylusTraceBuffer {
    pub fn new() -> Self {
        Self::default()
    }

    /// Append a single host-I/O record.
    pub fn push(&self, record: HostioTraceInfo) {
        if let Ok(mut g) = self.inner.lock() {
            g.push(record);
        }
    }

    /// Drain + return the collected records.
    pub fn drain(&self) -> Vec<HostioTraceInfo> {
        self.inner
            .lock()
            .map(|mut g| std::mem::take(&mut *g))
            .unwrap_or_default()
    }

    /// Clear the buffer.
    pub fn clear(&self) {
        if let Ok(mut g) = self.inner.lock() {
            g.clear();
        }
    }

    /// Number of records currently buffered.
    pub fn len(&self) -> usize {
        self.inner.lock().map(|g| g.len()).unwrap_or(0)
    }

    /// Whether the buffer has any records.
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

/// Top-level tracer output attached to a `debug_traceTransaction`
/// result when the transaction invoked a Stylus contract.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StylusTraceOutput {
    /// Flat list of host-I/O records captured during tx execution.
    pub hostio_records: Vec<HostioTraceInfo>,
}

impl From<Vec<HostioTraceInfo>> for StylusTraceOutput {
    fn from(hostio_records: Vec<HostioTraceInfo>) -> Self {
        Self { hostio_records }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn mk(name: &str, start_ink: u64, end_ink: u64) -> HostioTraceInfo {
        HostioTraceInfo {
            name: name.to_string(),
            args: Bytes::new(),
            outs: Bytes::new(),
            start_ink,
            end_ink,
            address: None,
            steps: Vec::new(),
        }
    }

    #[test]
    fn buffer_default_empty() {
        let b = StylusTraceBuffer::new();
        assert!(b.is_empty());
        assert_eq!(b.len(), 0);
    }

    #[test]
    fn buffer_push_and_drain() {
        let b = StylusTraceBuffer::new();
        b.push(mk("storage_load_bytes32", 100, 50));
        b.push(mk("contract_call", 50, 10));
        assert_eq!(b.len(), 2);
        let drained = b.drain();
        assert_eq!(drained.len(), 2);
        assert_eq!(drained[0].name, "storage_load_bytes32");
        assert!(b.is_empty());
    }

    #[test]
    fn buffer_clear() {
        let b = StylusTraceBuffer::new();
        b.push(mk("emit_log", 200, 150));
        b.clear();
        assert!(b.is_empty());
    }

    #[test]
    fn buffer_clone_shares_inner() {
        let b1 = StylusTraceBuffer::new();
        let b2 = b1.clone();
        b1.push(mk("getCaller", 10, 9));
        assert_eq!(b2.len(), 1);
    }

    #[test]
    fn hostio_serde_roundtrips() {
        let r = HostioTraceInfo {
            name: "contract_call".to_string(),
            args: Bytes::from(vec![0xDE, 0xAD]),
            outs: Bytes::from(vec![0xBE, 0xEF]),
            start_ink: 1_000,
            end_ink: 500,
            address: Some(Address::repeat_byte(0xAB)),
            steps: Vec::new(),
        };
        let json = serde_json::to_string(&r).unwrap();
        let back: HostioTraceInfo = serde_json::from_str(&json).unwrap();
        assert_eq!(back.name, r.name);
        assert_eq!(back.start_ink, r.start_ink);
        assert_eq!(back.address, r.address);
    }

    #[test]
    fn nested_steps_supported() {
        let mut parent = mk("contract_call", 1_000, 400);
        parent.steps.push(mk("storage_load_bytes32", 900, 800));
        parent.steps.push(mk("emit_log", 800, 600));
        assert_eq!(parent.steps.len(), 2);
    }
}
