//! Per-opcode multi-gas attribution for the v60 multi-dimensional pricing
//! model. Mirrors Nitro's per-opcode dimension assignment so the per-dimension
//! L2 pricing backlogs (and hence the base fee) match.

pub mod classify;
pub mod inspector;
pub mod intrinsic;

pub use classify::{OpKind, classify};
pub use inspector::{MultiGasInspector, MultiGasSink};
pub use intrinsic::{IntrinsicInput, intrinsic_multigas};
