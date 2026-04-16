//! Test fixtures and harness for the arbreth workspace.
//!
//! Mirrors Nitro's `storage.NewMemoryBacked(burn.NewSystemBurner(...))` pattern
//! so unit tests can exercise ArbOS state in-memory without wiring up a node.
//!
//! # Quick start
//!
//! ```no_run
//! use arb_test_utils::ArbosHarness;
//!
//! let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
//! let l1 = h.l1_pricing_state();
//! let last_update = l1.last_update_time().unwrap();
//! assert_eq!(last_update, 0);
//! ```

pub mod accounts;
pub mod db;
pub mod harness;

pub use accounts::{alice, bob, charlie, dave, eve, frank, test_account};
pub use db::EmptyDb;
pub use harness::ArbosHarness;
