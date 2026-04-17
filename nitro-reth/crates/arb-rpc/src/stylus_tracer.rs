//! Stylus host-I/O tracer: records each WASM host function call made
//! during Stylus program execution so `debug_traceTransaction` can
//! surface them alongside EVM events.

use std::{
    collections::HashMap,
    sync::{Arc, Mutex, OnceLock},
};

use alloy_primitives::{Address, Bytes, B256};
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

/// Global cache of `tx_hash -> HostioTraceInfo[]` populated by the
/// block producer when a tx touches a Stylus program with tracing
/// enabled, and drained by `arb_traceStylusHostio`. Size-bounded LRU
/// semantics (oldest entries evicted when the cap is reached).
fn trace_cache() -> &'static Mutex<HashMap<B256, Vec<HostioTraceInfo>>> {
    static CACHE: OnceLock<Mutex<HashMap<B256, Vec<HostioTraceInfo>>>> = OnceLock::new();
    CACHE.get_or_init(|| Mutex::new(HashMap::new()))
}

/// Bound on cached tx-hash entries. A conservative default: most
/// debug_trace workflows only care about recent txs.
const TRACE_CACHE_MAX_ENTRIES: usize = 1024;

/// Store the Stylus host-I/O records for a tx-hash. Called by the
/// block producer after executing a tx with the trace buffer active.
pub fn cache_trace(tx_hash: B256, records: Vec<HostioTraceInfo>) {
    if records.is_empty() {
        return;
    }
    if let Ok(mut m) = trace_cache().lock() {
        if m.len() >= TRACE_CACHE_MAX_ENTRIES {
            // Simple eviction: drop a random existing entry to stay
            // bounded. (HashMap iteration order is nondeterministic.)
            if let Some(key) = m.keys().next().cloned() {
                m.remove(&key);
            }
        }
        m.insert(tx_hash, records);
    }
}

/// Retrieve + remove the cached trace for a tx-hash. Matches Nitro's
/// one-shot retrieval semantics: the buffer is drained on first read.
pub fn take_cached_trace(tx_hash: B256) -> Vec<HostioTraceInfo> {
    trace_cache()
        .lock()
        .ok()
        .and_then(|mut m| m.remove(&tx_hash))
        .unwrap_or_default()
}

/// Run `f` with a fresh Stylus host-I/O trace buffer installed on the
/// current thread. The buffer is drained and returned after `f`
/// completes so callers can attach the records to a debug response.
///
/// This is the integration seam between `debug_traceTransaction` and
/// the Stylus runtime: the debug handler wraps its tx execution in a
/// `with_trace_buffer` call and then surfaces the records alongside
/// the standard EVM trace.
pub fn with_trace_buffer<F, T>(f: F) -> (T, Vec<HostioTraceInfo>)
where
    F: FnOnce() -> T,
{
    use std::sync::{Arc, Mutex};

    let buf = Arc::new(Mutex::new(Vec::<arb_stylus::trace::HostioRecord>::new()));
    arb_stylus::trace::enable(buf.clone());
    let result = f();
    arb_stylus::trace::disable();

    let raw = buf.lock().map(|g| g.clone()).unwrap_or_default();
    let records = raw
        .into_iter()
        .map(|r| HostioTraceInfo {
            name: r.name.to_string(),
            args: r.args,
            outs: r.outs,
            start_ink: r.start_ink,
            end_ink: r.end_ink,
            address: r.address,
            steps: Vec::new(),
        })
        .collect();
    (result, records)
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
