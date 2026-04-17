//! Host-I/O trace buffer for `debug_traceTransaction` on Stylus programs.
//!
//! Host functions call [`record`] to push a trace entry; a tracing
//! driver installs a buffer via [`enable`] before running the program
//! and [`take`]s the records afterwards.

use std::{
    cell::RefCell,
    sync::{Arc, Mutex},
};

use alloy_primitives::{Address, Bytes};

/// Single recorded host-I/O call.
#[derive(Debug, Clone)]
pub struct HostioRecord {
    pub name: &'static str,
    pub args: Bytes,
    pub outs: Bytes,
    pub start_ink: u64,
    pub end_ink: u64,
    pub address: Option<Address>,
    /// Sub-frame records for CALL/CREATE family. Empty for leaf hostios.
    pub steps: Vec<HostioRecord>,
}

thread_local! {
    static ACTIVE: RefCell<Option<Arc<Mutex<Vec<HostioRecord>>>>> = const { RefCell::new(None) };
    /// Stack of sub-call frames. While non-empty, recording goes into the
    /// top frame instead of the active buffer; the parent CALL/CREATE
    /// hostio attaches the popped frame as its `steps` field.
    static FRAMES: RefCell<Vec<Vec<HostioRecord>>> = const { RefCell::new(Vec::new()) };
}

/// Push a fresh sub-call frame. Subsequent [`record`] calls (until the
/// matching [`exit_subcall`]) accumulate inside this frame.
pub fn enter_subcall() {
    FRAMES.with(|f| f.borrow_mut().push(Vec::new()));
}

/// Pop the top sub-call frame and return its accumulated records. The
/// parent CALL/CREATE hostio attaches this list as its `steps`.
pub fn exit_subcall() -> Vec<HostioRecord> {
    FRAMES.with(|f| f.borrow_mut().pop().unwrap_or_default())
}

/// Install a buffer for the current thread. Subsequent [`record`]
/// calls append to it until [`disable`] is called.
pub fn enable(buf: Arc<Mutex<Vec<HostioRecord>>>) {
    ACTIVE.with(|slot| *slot.borrow_mut() = Some(buf));
}

/// Clear the active buffer for the current thread.
pub fn disable() {
    ACTIVE.with(|slot| *slot.borrow_mut() = None);
}

/// Take and clear the active buffer's contents.
pub fn take() -> Vec<HostioRecord> {
    ACTIVE
        .with(|slot| {
            slot.borrow()
                .as_ref()
                .and_then(|b| b.lock().ok().map(|mut v| std::mem::take(&mut *v)))
        })
        .unwrap_or_default()
}

/// Push one record into the active buffer (or the open sub-frame, if
/// any). A no-op when tracing is disabled — zero cost on the hot path.
pub fn record(
    name: &'static str,
    args: Bytes,
    outs: Bytes,
    start_ink: u64,
    end_ink: u64,
    address: Option<Address>,
) {
    record_with_steps(name, args, outs, start_ink, end_ink, address, Vec::new());
}

/// Like [`record`] but with pre-collected sub-frame records attached
/// as `steps` (used by CALL/CREATE family hostios after popping their
/// own sub-frame).
pub fn record_with_steps(
    name: &'static str,
    args: Bytes,
    outs: Bytes,
    start_ink: u64,
    end_ink: u64,
    address: Option<Address>,
    steps: Vec<HostioRecord>,
) {
    let rec = HostioRecord {
        name,
        args,
        outs,
        start_ink,
        end_ink,
        address,
        steps,
    };
    let leftover = FRAMES.with(|f| {
        let mut frames = f.borrow_mut();
        if let Some(top) = frames.last_mut() {
            top.push(rec);
            None
        } else {
            Some(rec)
        }
    });
    if let Some(rec) = leftover {
        ACTIVE.with(|slot| {
            if let Some(buf) = slot.borrow().as_ref() {
                if let Ok(mut v) = buf.lock() {
                    v.push(rec);
                }
            }
        });
    }
}

/// Whether tracing is active on the current thread — cheap check the
/// host functions can use to avoid building args/outs when disabled.
pub fn is_active() -> bool {
    ACTIVE.with(|slot| slot.borrow().is_some())
}

/// Convenience wrapper for host functions that want to record the
/// call name with optional args + outs and no ink delta (e.g., leaf
/// host functions that never block or touch state).
#[inline]
pub fn record_leaf(name: &'static str, args: Bytes, outs: Bytes) {
    if is_active() {
        record(name, args, outs, 0, 0, None);
    }
}

/// Record a host-function call with an ink delta captured by the
/// caller. Used where args/outs aren't meaningful but ink cost is.
#[inline]
pub fn record_ink(name: &'static str, start_ink: u64, end_ink: u64) {
    if is_active() {
        record(name, Bytes::new(), Bytes::new(), start_ink, end_ink, None);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn record_when_disabled_is_noop() {
        disable();
        assert!(!is_active());
        record("x", Bytes::new(), Bytes::new(), 100, 50, None);
        assert!(take().is_empty());
    }

    #[test]
    fn enable_captures_records() {
        let buf = Arc::new(Mutex::new(Vec::new()));
        enable(buf.clone());
        assert!(is_active());
        record(
            "storage_load_bytes32",
            Bytes::from(vec![1]),
            Bytes::from(vec![2]),
            100,
            90,
            None,
        );
        record(
            "contract_call",
            Bytes::new(),
            Bytes::new(),
            90,
            40,
            Some(Address::repeat_byte(0xAA)),
        );
        let records = take();
        assert_eq!(records.len(), 2);
        assert_eq!(records[0].name, "storage_load_bytes32");
        assert_eq!(records[1].address, Some(Address::repeat_byte(0xAA)));
        disable();
    }

    #[test]
    fn subcall_frame_nests_records() {
        let buf = Arc::new(Mutex::new(Vec::new()));
        enable(buf.clone());

        // Top-level record before sub-call.
        record(
            "storage_load_bytes32",
            Bytes::new(),
            Bytes::new(),
            100,
            90,
            None,
        );

        // Sub-call: enter, record two inner hostios, exit, record parent.
        enter_subcall();
        record(
            "storage_load_bytes32",
            Bytes::new(),
            Bytes::new(),
            80,
            70,
            None,
        );
        record("emit_log", Bytes::new(), Bytes::new(), 70, 60, None);
        let steps = exit_subcall();
        assert_eq!(steps.len(), 2);
        record_with_steps(
            "call_contract",
            Bytes::new(),
            Bytes::new(),
            85,
            55,
            Some(Address::repeat_byte(0xCC)),
            steps,
        );

        let records = take();
        assert_eq!(records.len(), 2);
        assert_eq!(records[0].name, "storage_load_bytes32");
        assert_eq!(records[0].steps.len(), 0);
        assert_eq!(records[1].name, "call_contract");
        assert_eq!(records[1].steps.len(), 2);
        assert_eq!(records[1].steps[0].name, "storage_load_bytes32");
        assert_eq!(records[1].steps[1].name, "emit_log");
        disable();
    }

    #[test]
    fn nested_subcalls_compose() {
        let buf = Arc::new(Mutex::new(Vec::new()));
        enable(buf.clone());
        enter_subcall();
        record("a", Bytes::new(), Bytes::new(), 0, 0, None);
        enter_subcall();
        record("b", Bytes::new(), Bytes::new(), 0, 0, None);
        record("c", Bytes::new(), Bytes::new(), 0, 0, None);
        let inner = exit_subcall();
        assert_eq!(inner.len(), 2);
        record_with_steps("inner_call", Bytes::new(), Bytes::new(), 0, 0, None, inner);
        record("d", Bytes::new(), Bytes::new(), 0, 0, None);
        let outer = exit_subcall();
        assert_eq!(outer.len(), 3);
        assert_eq!(outer[1].name, "inner_call");
        assert_eq!(outer[1].steps.len(), 2);
        record_with_steps("outer_call", Bytes::new(), Bytes::new(), 0, 0, None, outer);
        let records = take();
        assert_eq!(records.len(), 1);
        assert_eq!(records[0].name, "outer_call");
        assert_eq!(records[0].steps.len(), 3);
        disable();
    }

    #[test]
    fn take_clears_buffer() {
        let buf = Arc::new(Mutex::new(Vec::new()));
        enable(buf);
        record("foo", Bytes::new(), Bytes::new(), 10, 5, None);
        assert_eq!(take().len(), 1);
        assert_eq!(take().len(), 0);
        disable();
    }
}
