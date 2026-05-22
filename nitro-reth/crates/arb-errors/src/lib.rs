//! Top-level error types for arbreth.
//!
//! Thin umbrella over the leaf-level error crates (currently
//! `arb-storage-errors`), mirroring `reth-errors`. Downstream crates that
//! cross multiple subsystem boundaries depend on this crate and convert
//! into [`ArbError`] via `?`, instead of being generic over every leaf
//! error type.

#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

mod error;
pub use error::{ArbError, ArbResult};

pub use arb_storage_errors::{DatabaseError, DatabaseErrorInfo, StorageError};
