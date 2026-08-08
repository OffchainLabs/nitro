//! Error types for the arb-storage layer.
//!
//! This crate intentionally mirrors `reth-storage-errors`: a thin leaf-level
//! crate that erases the underlying `Database::Error` into a concrete
//! [`DatabaseError`] so downstream consumers do not have to be generic over
//! it. The wider [`StorageError`] adds storage-layout-specific failure modes
//! (decode overflow, invalid layout, broken invariant) on top.

#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(not(feature = "std"), no_std)]

extern crate alloc;

mod any;
mod db;
mod storage;

pub use any::AnyError;
pub use db::{DatabaseError, DatabaseErrorInfo};
pub use storage::StorageError;
