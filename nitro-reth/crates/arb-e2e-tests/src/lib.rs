//! End-to-end tests for the Arbitrum block executor.
//!
//! Lives in its own crate to break the dep cycle between arb-evm and
//! arb-node — both are needed for a real e2e flow.
