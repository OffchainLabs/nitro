// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

pub mod binary;
pub mod cbytes;
mod host;
pub(crate) mod internal_func;
#[cfg(feature = "native")]
mod kzg;
pub mod machine;
/// cbindgen:ignore
pub mod memory;
pub(crate) mod memory_type;
pub mod merkle;
pub mod prepare;
mod print;
pub mod programs;
mod reinterpret;
pub mod utils;
pub mod value;
pub mod wavm;

#[cfg(test)]
mod test;

pub use machine::Machine;
