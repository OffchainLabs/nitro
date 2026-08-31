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

#[test]
fn set_pages_seeds_open_and_ever_for_subcall() {
    let env = PageTracker {
        open: 3,
        ever: 7,
        free_pages: 2,
        page_gas: 1_000,
        page_limit: 0,
    };
    assert_eq!(env.open, 3);
    assert_eq!(env.ever, 7);
    assert_eq!(env.free_pages, 2);
    assert_eq!(env.page_gas, 1_000);
}

#[test]
fn page_charge_advances_open_and_ever_for_fresh_env() {
    let mut env = PageTracker {
        open: 0,
        ever: 0,
        free_pages: 0,
        page_gas: 100,
        page_limit: 0,
    };
    let arbos_version = 0;
    env.charge(5, arbos_version);
    assert_eq!(env.open, 5);
    assert_eq!(env.ever, 5);
}

#[test]
fn page_charge_accumulates_open() {
    let mut env = PageTracker {
        open: 0,
        ever: 0,
        free_pages: 0,
        page_gas: 100,
        page_limit: 0,
    };
    let arbos_version = 0;
    env.charge(5, arbos_version);
    env.charge(3, arbos_version);
    assert_eq!(env.open, 8);
    assert_eq!(env.ever, 8);
}

#[test]
fn page_charge_saturates_on_overflow() {
    let mut env = PageTracker {
        open: u16::MAX - 5,
        ever: u16::MAX - 5,
        free_pages: 0,
        page_gas: 0,
        page_limit: 0,
    };
    let arbos_version = 0;
    env.charge(100, arbos_version);
    assert_eq!(env.open, u16::MAX);
    assert_eq!(env.ever, u16::MAX);
}

#[test]
fn pages_ever_is_high_water_mark_after_freeing() {
    let mut env = PageTracker {
        open: 0,
        ever: 0,
        free_pages: 0,
        page_gas: 100,
        page_limit: 0,
    };
    let arbos_version = 0;
    env.charge(10, arbos_version);
    // Simulate a sub-call freeing memory by writing the lower open count back.
    env.open = 2;
    assert_eq!(env.ever, 10);
    // Re-allocating beyond the previous high-water mark advances ever.
    env.charge(20, arbos_version);
    assert_eq!(env.open, 22);
    assert_eq!(env.ever, 22);
}

#[test]
fn page_charge_below_free_pages_is_free() {
    let mut env = PageTracker {
        open: 0,
        ever: 0,
        free_pages: 4,
        page_gas: 500,
        page_limit: 0,
    };
    let arbos_version = 0;
    let cost = env.charge(3, arbos_version); // still within free window
    assert_eq!(cost, 0);
    assert_eq!(env.open, 3);
    assert_eq!(env.ever, 3);
}

#[test]
fn page_charge_matches_memory_model_for_paid_pages() {
    let mut env = PageTracker {
        open: 0,
        ever: 0,
        free_pages: 2,
        page_gas: 1_000,
        page_limit: 0,
    };
    let arbos_version = 0;
    let cost = env.charge(5, arbos_version);
    let expected = MemoryModel::new(2, 1_000).gas_cost(5, 0, 0);
    assert_eq!(cost, expected);
}

// ── Consensus page limit gate (ArbOS >= 59) ─────────────────────────

use arb_stylus::evm_api_impl::{PageTracker, page_limit_exceeded};

#[test]
fn page_limit_gate_inert_before_v59() {
    assert!(!page_limit_exceeded(58, 4, 9));
}

#[test]
fn page_limit_gate_active_at_v59_and_v60() {
    assert!(page_limit_exceeded(59, 4, 9));
    assert!(page_limit_exceeded(60, 4, 9));
}

#[test]
fn page_limit_gate_inert_when_limit_zero() {
    assert!(!page_limit_exceeded(60, 0, 9));
}

#[test]
fn page_limit_gate_boundary_is_strict() {
    assert!(!page_limit_exceeded(60, 9, 9));
    assert!(page_limit_exceeded(60, 8, 9));
}

#[test]
fn page_charge_saturates_over_limit_at_v60() {
    let mut env = PageTracker {
        open: 1,
        ever: 1,
        free_pages: 0,
        page_gas: 100,
        page_limit: 4,
    };
    let arbos_version = 60;
    assert_eq!(env.charge(8, arbos_version), u64::MAX);
    assert_eq!(env.open, 9);
}

#[test]
fn page_charge_finite_over_limit_at_v58() {
    let mut env = PageTracker {
        open: 1,
        ever: 1,
        free_pages: 0,
        page_gas: 100,
        page_limit: 4,
    };
    let arbos_version = 58;
    assert_ne!(env.charge(8, arbos_version), u64::MAX);
    assert_eq!(env.open, 9);
}

#[test]
fn page_charge_finite_under_limit_at_v60() {
    let mut env = PageTracker {
        open: 1,
        ever: 1,
        free_pages: 0,
        page_gas: 100,
        page_limit: 128,
    };
    let arbos_version = 60;
    let cost = env.charge(8, arbos_version);
    assert_ne!(cost, u64::MAX);
    assert_eq!(cost, MemoryModel::new(0, 100).gas_cost(8, 1, 1));
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
