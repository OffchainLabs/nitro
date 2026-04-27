//! Shared test orchestration for arbreth.
//!
//! This crate provides a clean Rust abstraction for:
//!  - Spawning arbreth and a Nitro reference node ([`node`]).
//!  - Building L1 messages of every kind ([`messaging`]).
//!  - Composing genesis chain specs from [`arb_chainspec`] types ([`genesis`]).
//!  - Running a [`scenario::Scenario`] against one or two nodes
//!    ([`dual_exec`]) and producing a structural [`dual_exec::DiffReport`].
//!  - Capturing node state into the [`crate::capture::CapturedScenario`]
//!    that [`arb-spec-tests`] consumes as an `ExecutionFixture`.
//!
//! The same abstractions back the spec-test runner, the differential
//! fuzz targets, and the operator CLI. There is exactly one place where
//! "spawn a node" or "build a deposit message" is implemented.

pub mod capture;
pub mod dual_exec;
pub mod error;
pub mod genesis;
pub mod messaging;
pub mod mock_l1;
pub mod node;
pub mod rpc;
pub mod scenario;
#[cfg(feature = "stylus-wat")]
pub mod stylus;

pub use error::{HarnessError, Result};

pub use dual_exec::{BlockDiff, DiffReport, DualExec, LogDiff, StateDiff, TxDiff};
pub use messaging::L1Message;
pub use node::{ArbReceiptFields, BlockId, ExecutionNode, NodeKind, NodeStartCtx};
pub use scenario::{Scenario, ScenarioStep};
