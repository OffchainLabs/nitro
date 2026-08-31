//! Page accounting and reentrancy tests for the Stylus runtime.
//!
//! After the migration off thread-local globals, page tracking lives on
//! `WasmEnv` (per-instance, threaded across sub-calls explicitly) and the
//! reentrancy counter lives on `TxCtx`. These tests cover both surfaces.

#[cfg(target_arch = "x86_64")]
#[unsafe(no_mangle)]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use alloy_primitives::{Address, address};
use arb_context::ArbPrecompileCtx;
use arbos::programs::memory::MemoryModel;

// ── MemoryModel: shared between WasmEnv and the precompile path ─────

#[test]
fn memory_model_returns_zero_below_free_pages() {
    let model = MemoryModel::new(2, 1_000);
    assert_eq!(model.gas_cost(2, 0, 0), 0);
}

#[test]
fn memory_model_charges_linear_plus_exponential_beyond_free() {
    let model = MemoryModel::new(2, 1_000);
    // open=0, ever=0, allocate 4 pages: 2 free + 2 paid linear + exp(4)-exp(0).
    let cost_first = model.gas_cost(4, 0, 0);
    assert!(cost_first > 0);
    // Re-allocating without freeing piles on more cost.
    let cost_second = model.gas_cost(2, 4, 4);
    assert!(cost_second > 0);
}

#[test]
fn memory_model_reuses_ever_high_water_mark_when_reopening() {
    let model = MemoryModel::new(0, 100);
    let full = model.gas_cost(10, 0, 0);
    // Closing and reopening the same pages skips the exponential delta because
    // `ever` already covers them; only the linear portion is charged.
    let reopen = model.gas_cost(10, 0, 10);
    assert!(reopen < full);
}

// ── pay_for_memory_grow operand width (ArbOS >= 59) ─────────────────
//
// The hostio param is u32; an operand wider than u16::MAX buys the whole budget
// (OOG) before truncation at ArbOS >= 59. Below the activation version, or
// within u16 range, the gate is inert.

use arb_stylus::env::pay_for_memory_grow_overflows;

#[test]
fn pay_for_memory_grow_width_gate() {
    assert!(pay_for_memory_grow_overflows(60, 65_536)); // u16::MAX + 1 at v60
    assert!(pay_for_memory_grow_overflows(59, u32::MAX));
    assert!(!pay_for_memory_grow_overflows(60, u32::from(u16::MAX))); // exactly u16::MAX
    assert!(!pay_for_memory_grow_overflows(60, 16)); // small grow
    assert!(!pay_for_memory_grow_overflows(58, 65_536)); // pre-activation version
}

// ── Reentrancy counter (now on TxCtx via ArbPrecompileCtx) ──────────

const PROG_A: Address = address!("aaaa000000000000000000000000000000000000");
const PROG_B: Address = address!("bbbb000000000000000000000000000000000000");

#[test]
fn push_stylus_program_signals_reentrancy_on_second_entry() {
    let ctx = ArbPrecompileCtx::default();
    assert!(!ctx.push_stylus_program(PROG_A));
    assert!(ctx.push_stylus_program(PROG_A));
    assert!(ctx.push_stylus_program(PROG_A));
}

#[test]
fn push_distinct_addresses_are_not_reentrant() {
    let ctx = ArbPrecompileCtx::default();
    assert!(!ctx.push_stylus_program(PROG_A));
    assert!(!ctx.push_stylus_program(PROG_B));
}

#[test]
fn pop_stylus_program_decrements_then_removes_at_zero() {
    let ctx = ArbPrecompileCtx::default();
    ctx.push_stylus_program(PROG_A);
    ctx.push_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 2);
    ctx.pop_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 1);
    ctx.pop_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 0);
    // Extra pops saturate without panicking.
    ctx.pop_stylus_program(PROG_A);
    assert_eq!(ctx.stylus_program_count(PROG_A), 0);
}
