//! Single-node tests for the Stylus consensus `PageLimit` gate and the
//! `pay_for_memory_grow` operand width, asserted against the runtime primitives
//! directly: the page charge saturates exactly when `arbos_version >= 59 &&
//! page_limit > 0 && new_open > page_limit`, and is inert at
//! `arbos_version == 58`.

#[cfg(target_arch = "x86_64")]
#[unsafe(no_mangle)]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use arb_stylus::{env::page_limit_exceeded, evm_api_impl::PageTracker};
use arbos::programs::memory::MemoryModel;

const ARBOS_59: u64 = 59;
const ARBOS_58: u64 = 58;
const ARBOS_60: u64 = 60;

const FREE_PAGES: u16 = 2;
const PAGE_GAS: u16 = 1_000;

// ── page_limit_exceeded truth table ─────────────────────────────────

#[test]
fn predicate_fires_only_above_limit_at_v59_plus() {
    assert!(page_limit_exceeded(ARBOS_60, 4, 9));
    assert!(page_limit_exceeded(ARBOS_59, 4, 5));
    assert!(!page_limit_exceeded(ARBOS_60, 4, 4));
    assert!(!page_limit_exceeded(ARBOS_60, 4, 3));
}

#[test]
fn predicate_inert_below_v59() {
    assert!(!page_limit_exceeded(ARBOS_58, 4, 9));
    assert!(!page_limit_exceeded(0, 4, 9));
}

#[test]
fn predicate_inert_when_limit_zero() {
    assert!(!page_limit_exceeded(ARBOS_60, 0, 9));
}

// ── page-charge saturation ─────────────────────────────────────

fn pages_with(open: u16, page_limit: u16) -> PageTracker {
    PageTracker {
        open,
        ever: open,
        free_pages: FREE_PAGES,
        page_gas: PAGE_GAS,
        page_limit,
    }
}

/// Footprint 1 already open, grow 8 → open 9 > limit 4 at arbos 60: the charge
/// saturates to `u64::MAX`.
#[test]
fn page_charge_saturates_over_limit_at_v60() {
    let mut env = pages_with(1, 4);
    let cost = env.charge(8, ARBOS_60);
    assert_eq!(cost, u64::MAX);
    assert_eq!(env.open, 9);
}

/// Same open/grow/limit, arbos 58: the gate is inert, so the charge is the
/// ordinary finite memory-model cost.
#[test]
fn page_charge_inert_below_v59() {
    let mut env = pages_with(1, 4);
    let cost = env.charge(8, ARBOS_58);
    let expected = MemoryModel::new(FREE_PAGES, PAGE_GAS).gas_cost(8, 1, 1);
    assert_eq!(cost, expected);
    assert!(cost < u64::MAX);
    assert_eq!(env.open, 9);
}

/// Exactly at the limit is allowed (`new_open > page_limit` is strict).
#[test]
fn page_charge_exactly_at_limit_is_finite() {
    let mut env = pages_with(1, 9);
    let cost = env.charge(8, ARBOS_60);
    assert!(cost < u64::MAX);
    assert_eq!(env.open, 9);
}

/// A zero `page_limit` disables the cap even at arbos 60.
#[test]
fn page_charge_zero_limit_disabled() {
    let mut env = pages_with(1, 0);
    let cost = env.charge(8, ARBOS_60);
    assert!(cost < u64::MAX);
}

/// Open 9 stays under the ample default limit 128, so the charge is finite at
/// arbos 60.
#[test]
fn page_charge_under_default_limit_is_finite() {
    let mut env = pages_with(1, 128);
    let cost = env.charge(8, ARBOS_60);
    assert!(cost < u64::MAX);
    assert_eq!(env.open, 9);
}

// ── entry-footprint site (stylus_call_gas_cost in arb-evm) ──────────

/// The entry-footprint reservation adds `u64::MAX` when `pages_open + footprint`
/// exceeds the limit at arbos >= 59, for a program that never grows at runtime.
#[test]
fn entry_footprint_predicate_matches_call_site() {
    assert!(page_limit_exceeded(ARBOS_60, 4, 0u16.saturating_add(5)));
    assert!(!page_limit_exceeded(ARBOS_58, 4, 0u16.saturating_add(5)));
    assert!(!page_limit_exceeded(ARBOS_60, 5, 0u16.saturating_add(5)));
}
