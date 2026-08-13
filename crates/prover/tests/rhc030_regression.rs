// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! CI-safe regression guard for the RHC-030 Stylus-parser locals memory bomb.
//!
//! The bug: `parse_with_stylus_version` materialized every declared wasm local
//! into an 8-byte `Local` object *before* the anti-DoS limit ran, so a
//! size-legal module (compact `(count, i32)` groups) could allocate multiple GB
//! during activation. The fix rejects excess locals *during* parsing, on the
//! `parse_user` (activation) path.
//!
//! This test drives the guarded path and asserts BOTH properties that matter:
//!   1. rejection happens (error mentions "too many wasm locals"), and
//!   2. it happens BEFORE the expansion — peak allocation stays bounded.
//!
//! Property 2 is what a plain "limit exists" test misses: if the check ever
//! regresses to *after* materialization, this module allocates ~0.26 GB.
//! A capped `#[global_allocator]` aborts past `CAP` so a regression fails fast
//! instead of OOMing the runner; with the fix present, parse never approaches
//! the cap. Lives in its own integration-test binary so the allocator is
//! isolated from other tests.

use std::{
    alloc::{GlobalAlloc, Layout, System},
    io::Write,
    sync::atomic::{AtomicBool, AtomicUsize, Ordering::Relaxed},
};

use arbutil::{Bytes32, evm::ARBOS_VERSION_STYLUS_CHARGING_FIXES};
use prover::{binary::WasmBinary, programs::config::CompileConfig};

/// Catastrophic safety net: abort before a regression can truly OOM the runner.
/// Set well above the ~0.26 GB a locals-limit regression allocates at this test's
/// scale, so that case fails via the clean `BOUND` assertion instead; the cap
/// only catches larger (e.g. bigger-scale or multi-limit) blow-ups.
const CAP: usize = 2 * 1024 * 1024 * 1024;
/// Assertion ceiling: the fix must keep parse comfortably under this (~1.5 MB).
const BOUND: usize = 64 * 1024 * 1024;

struct Capped;
static CUR: AtomicUsize = AtomicUsize::new(0);
static PEAK: AtomicUsize = AtomicUsize::new(0);
static ABORTING: AtomicBool = AtomicBool::new(false);

unsafe impl GlobalAlloc for Capped {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        let now = CUR.fetch_add(layout.size(), Relaxed) + layout.size();
        PEAK.fetch_max(now, Relaxed);
        if now > CAP {
            // Reentrancy guard: the message write itself allocates, which would
            // re-enter here; the first tripper writes, re-entry short-circuits
            // straight to abort (no recursion / stack overflow).
            if !ABORTING.swap(true, Relaxed) {
                let _ = std::io::stderr()
                    .write_all(b"RHC-030 REGRESSION: parse exceeded memory cap before rejecting\n");
            }
            std::process::abort();
        }
        unsafe { System.alloc(layout) }
    }
    unsafe fn dealloc(&self, ptr: *mut u8, layout: Layout) {
        unsafe { System.dealloc(ptr, layout) };
        CUR.fetch_sub(layout.size(), Relaxed);
    }
}

#[global_allocator]
static GLOBAL: Capped = Capped;

fn leb128_u32(mut v: u32, out: &mut Vec<u8>) {
    loop {
        let mut byte = (v & 0x7f) as u8;
        v >>= 7;
        if v != 0 {
            byte |= 0x80;
        }
        out.push(byte);
        if v == 0 {
            break;
        }
    }
}

fn wasm_section(id: u8, body: &[u8], out: &mut Vec<u8>) {
    out.push(id);
    leb128_u32(body.len() as u32, out);
    out.extend_from_slice(body);
}

/// `num_funcs` functions of type `() -> ()`, each declaring `locals_per_func`
/// i32 locals via one compact `(count, i32)` group and an empty body. The
/// compact count is the amplifier the bug relies on; neither `wat` nor
/// `wat2wasm` can emit it without listing every local.
fn build_locals_bomb(num_funcs: u32, locals_per_func: u32) -> Vec<u8> {
    let mut wasm = vec![0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00];

    // Type section: one type () -> ().
    wasm_section(0x01, &[0x01, 0x60, 0x00, 0x00], &mut wasm);

    // Function section: num_funcs entries, all type 0.
    let mut funcs = Vec::new();
    leb128_u32(num_funcs, &mut funcs);
    funcs.resize(funcs.len() + num_funcs as usize, 0x00);
    wasm_section(0x03, &funcs, &mut wasm);

    // Code section: num_funcs identical entries, each = 1 local group + `end`.
    let mut inner = vec![0x01]; // one local group
    leb128_u32(locals_per_func, &mut inner);
    inner.push(0x7f); // i32
    inner.push(0x0b); // end
    let mut entry = Vec::new();
    leb128_u32(inner.len() as u32, &mut entry);
    entry.extend_from_slice(&inner);

    let mut code = Vec::new();
    leb128_u32(num_funcs, &mut code);
    for _ in 0..num_funcs {
        code.extend_from_slice(&entry);
    }
    wasm_section(0x0a, &code, &mut wasm);

    wasm
}

/// 512 functions x 49_999 i32 locals (well under the 4096 function cap, so the
/// functions guard does not short-circuit — this exercises the *locals* guard).
/// Unfixed, `parse_user` materializes 512 * 49_999 = 25_599_488 `Local`s
/// (~0.26 GB with `Vec` growth) before rejecting; fixed, it bails at the first
/// function's locals group (~1.5 MB). The scale is chosen so a regression's peak
/// clears `BOUND` (fails the assertion cleanly) yet stays under `CAP` (no abort,
/// CI-safe). Passes iff the locals limit runs *before* expansion.
#[test]
fn parse_user_rejects_locals_before_expansion_bounded() {
    let wasm = build_locals_bomb(512, 49_999);
    assert!(
        wasm.len() < 128 * 1024,
        "payload must be size-legal ({} bytes)",
        wasm.len()
    );

    let compile = CompileConfig::version(1, false);

    // Absolute live-allocation peak across the process. Baseline (harness +
    // building the <128 KiB payload) is a few MB; the fixed parse adds ~1.5 MB;
    // a post-expansion regression adds ~0.26 GB (clears BOUND). We do NOT reset
    // the counter mid-run: that would let earlier allocations' `dealloc` drive
    // the running total below zero (usize underflow) and falsely trip the cap.
    let res = WasmBinary::parse_user(
        &wasm,
        1,
        ARBOS_VERSION_STYLUS_CHARGING_FIXES,
        u16::MAX,
        &compile,
        &Bytes32::default(),
    );
    let peak = PEAK.load(Relaxed);

    let err = res
        .err()
        .map(|e| e.to_string())
        .expect("locals bomb must be rejected");

    // Memory bound first: this is the property the bug violates. A regression
    // that keeps the limit but positions it after expansion trips exactly here.
    assert!(
        peak < BOUND,
        "parse allocated {peak} bytes (> {BOUND}); the locals check likely runs \
         AFTER expansion — RHC-030 regression (error was: {err})"
    );
    // And the rejection must be the early locals guard, not a later/unrelated one.
    assert!(
        err.contains("too many wasm locals"),
        "expected an early locals rejection, got: {err}"
    );

    eprintln!("OK: rejected \"{err}\"; peak live allocation = {peak} bytes (bound {BOUND})");
}
