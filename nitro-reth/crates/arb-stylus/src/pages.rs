//! WASM memory-page accounting for Stylus programs.
//!
//! Mirrors Go's `programs.memoryModel` bookkeeping: the EVM side of the API
//! owns the open/ever page counters, charges growth through the shared
//! `MemoryModel`, and enforces the consensus open-page cap (ArbOS >= 59).

use arb_chainspec::arbos_version::ARBOS_VERSION_59;
use arbos::programs::memory::MemoryModel;

/// Consensus open-page cap (ArbOS >= 59): a non-zero `page_limit` that
/// `new_open` exceeds makes the allocation unpayable.
pub fn page_limit_exceeded(arbos_version: u64, page_limit: u16, new_open: u16) -> bool {
    arbos_version >= ARBOS_VERSION_59 && page_limit > 0 && new_open > page_limit
}

/// WASM memory-page accounting: open/ever counters plus the memory-model parameters.
#[derive(Debug, Clone, Copy, Default)]
pub struct PageTracker {
    pub open: u16,
    pub ever: u16,
    pub free_pages: u16,
    pub page_gas: u16,
    pub page_limit: u16,
}

impl PageTracker {
    /// Charge for allocating `new_pages`, updating the open/ever counters and
    /// returning the gas cost.
    pub fn charge(&mut self, new_pages: u16, arbos_version: u64) -> u64 {
        let model = MemoryModel::new(self.free_pages, self.page_gas);
        let cost = model.gas_cost(new_pages, self.open, self.ever);
        self.open = self.open.saturating_add(new_pages);
        self.ever = self.ever.max(self.open);
        if page_limit_exceeded(arbos_version, self.page_limit, self.open) {
            return u64::MAX;
        }
        cost
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn tracker(
        open: u16,
        ever: u16,
        free_pages: u16,
        page_gas: u16,
        page_limit: u16,
    ) -> PageTracker {
        PageTracker {
            open,
            ever,
            free_pages,
            page_gas,
            page_limit,
        }
    }

    #[test]
    fn gate_inert_before_v59() {
        assert!(!page_limit_exceeded(58, 4, 9));
    }

    #[test]
    fn gate_active_at_v59_and_v60() {
        assert!(page_limit_exceeded(59, 4, 9));
        assert!(page_limit_exceeded(60, 4, 9));
    }

    #[test]
    fn gate_inert_when_limit_zero() {
        assert!(!page_limit_exceeded(60, 0, 9));
    }

    #[test]
    fn gate_boundary_is_strict() {
        assert!(!page_limit_exceeded(60, 9, 9));
        assert!(page_limit_exceeded(60, 8, 9));
    }

    #[test]
    fn charge_advances_open_and_ever() {
        let mut pages = tracker(0, 0, 0, 100, 0);
        pages.charge(5, 0);
        assert_eq!(pages.open, 5);
        assert_eq!(pages.ever, 5);
    }

    #[test]
    fn charge_accumulates_open() {
        let mut pages = tracker(0, 0, 0, 100, 0);
        pages.charge(5, 0);
        pages.charge(3, 0);
        assert_eq!(pages.open, 8);
        assert_eq!(pages.ever, 8);
    }

    #[test]
    fn charge_saturates_counters_on_overflow() {
        let mut pages = tracker(u16::MAX - 5, u16::MAX - 5, 0, 0, 0);
        pages.charge(100, 0);
        assert_eq!(pages.open, u16::MAX);
        assert_eq!(pages.ever, u16::MAX);
    }

    #[test]
    fn ever_is_high_water_mark_after_freeing() {
        let mut pages = tracker(0, 0, 0, 100, 0);
        pages.charge(10, 0);
        // Simulate a sub-call freeing memory by writing the lower open count back.
        pages.open = 2;
        assert_eq!(pages.ever, 10);
        // Re-allocating beyond the previous high-water mark advances ever.
        pages.charge(20, 0);
        assert_eq!(pages.open, 22);
        assert_eq!(pages.ever, 22);
    }

    #[test]
    fn charge_below_free_pages_is_free() {
        let mut pages = tracker(0, 0, 4, 500, 0);
        let cost = pages.charge(3, 0); // still within the free window
        assert_eq!(cost, 0);
        assert_eq!(pages.open, 3);
        assert_eq!(pages.ever, 3);
    }

    #[test]
    fn charge_matches_memory_model_for_paid_pages() {
        let mut pages = tracker(0, 0, 2, 1_000, 0);
        let cost = pages.charge(5, 0);
        assert_eq!(cost, MemoryModel::new(2, 1_000).gas_cost(5, 0, 0));
    }

    #[test]
    fn charge_saturates_over_limit_at_v60() {
        let mut pages = tracker(1, 1, 0, 100, 4);
        assert_eq!(pages.charge(8, 60), u64::MAX);
        assert_eq!(pages.open, 9);
    }

    #[test]
    fn charge_finite_over_limit_at_v58() {
        let mut pages = tracker(1, 1, 0, 100, 4);
        assert_ne!(pages.charge(8, 58), u64::MAX);
        assert_eq!(pages.open, 9);
    }

    #[test]
    fn charge_exactly_at_limit_is_finite() {
        let mut pages = tracker(1, 1, 0, 100, 9);
        assert!(pages.charge(8, 60) < u64::MAX);
        assert_eq!(pages.open, 9);
    }

    #[test]
    fn charge_zero_limit_disables_the_cap() {
        let mut pages = tracker(1, 1, 0, 100, 0);
        assert!(pages.charge(8, 60) < u64::MAX);
    }

    #[test]
    fn charge_finite_under_limit_at_v60() {
        let mut pages = tracker(1, 1, 0, 100, 128);
        let cost = pages.charge(8, 60);
        assert_ne!(cost, u64::MAX);
        assert_eq!(cost, MemoryModel::new(0, 100).gas_cost(8, 1, 1));
    }
}
