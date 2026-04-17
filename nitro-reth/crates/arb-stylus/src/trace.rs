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
}

thread_local! {
    static ACTIVE: RefCell<Option<Arc<Mutex<Vec<HostioRecord>>>>> = const { RefCell::new(None) };
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

/// Push one record into the active buffer. A no-op when tracing is
/// disabled — zero cost on the hot path.
pub fn record(
    name: &'static str,
    args: Bytes,
    outs: Bytes,
    start_ink: u64,
    end_ink: u64,
    address: Option<Address>,
) {
    ACTIVE.with(|slot| {
        if let Some(buf) = slot.borrow().as_ref() {
            if let Ok(mut v) = buf.lock() {
                v.push(HostioRecord {
                    name,
                    args,
                    outs,
                    start_ink,
                    end_ink,
                    address,
                });
            }
        }
    });
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
    fn take_clears_buffer() {
        let buf = Arc::new(Mutex::new(Vec::new()));
        enable(buf);
        record("foo", Bytes::new(), Bytes::new(), 10, 5, None);
        assert_eq!(take().len(), 1);
        assert_eq!(take().len(), 0);
        disable();
    }
}
