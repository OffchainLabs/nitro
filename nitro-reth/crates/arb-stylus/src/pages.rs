use alloy_primitives::Address;
use std::cell::Cell;
use std::cell::RefCell;
use std::collections::HashMap;

thread_local! {
    /// Currently open WASM memory pages across all active Stylus calls.
    static STYLUS_PAGES_OPEN: Cell<u16> = const { Cell::new(0) };
    /// High-water mark of pages ever open during this transaction.
    static STYLUS_PAGES_EVER: Cell<u16> = const { Cell::new(0) };
    /// Per-address count of open Stylus execution contexts (for reentrancy).
    static STYLUS_PROGRAM_COUNTS: RefCell<HashMap<Address, u32>> = RefCell::new(HashMap::new());
}

/// Reset Stylus page counters at transaction start.
pub fn reset_stylus_pages() {
    STYLUS_PAGES_OPEN.with(|v| v.set(0));
    STYLUS_PAGES_EVER.with(|v| v.set(0));
    STYLUS_PROGRAM_COUNTS.with(|v| v.borrow_mut().clear());
}

/// Get current (open, ever) page counts.
pub fn get_stylus_pages() -> (u16, u16) {
    let open = STYLUS_PAGES_OPEN.with(|v| v.get());
    let ever = STYLUS_PAGES_EVER.with(|v| v.get());
    (open, ever)
}

/// Add pages for a new Stylus call. Returns previous (open, ever).
pub fn add_stylus_pages(footprint: u16) -> (u16, u16) {
    let open = STYLUS_PAGES_OPEN.with(|v| v.get());
    let ever = STYLUS_PAGES_EVER.with(|v| v.get());
    let new_open = open.saturating_add(footprint);
    STYLUS_PAGES_OPEN.with(|v| v.set(new_open));
    STYLUS_PAGES_EVER.with(|v| v.set(ever.max(new_open)));
    (open, ever)
}

/// Restore page count after Stylus call returns.
pub fn set_stylus_pages_open(open: u16) {
    STYLUS_PAGES_OPEN.with(|v| v.set(open));
}

/// Push a Stylus program address onto the reentrancy tracker.
/// Returns true if this is a reentrant call (address was already active).
pub fn push_stylus_program(addr: Address) -> bool {
    STYLUS_PROGRAM_COUNTS.with(|v| {
        let mut map = v.borrow_mut();
        let count = map.entry(addr).or_insert(0);
        *count += 1;
        *count > 1
    })
}

/// Pop a Stylus program address from the reentrancy tracker.
pub fn pop_stylus_program(addr: Address) {
    STYLUS_PROGRAM_COUNTS.with(|v| {
        let mut map = v.borrow_mut();
        if let Some(count) = map.get_mut(&addr) {
            *count = count.saturating_sub(1);
            if *count == 0 {
                map.remove(&addr);
            }
        }
    });
}
