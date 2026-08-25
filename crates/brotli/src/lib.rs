// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

pub mod cgo;
mod dicts;
mod types;

#[cfg(feature = "wasmer_traits")]
mod wasmer_traits;

#[cfg(feature = "link")]
mod native;

#[cfg(feature = "link")]
pub use native::*;
pub use {
    dicts::Dictionary,
    types::{BrotliStatus, DEFAULT_WINDOW_SIZE}
};