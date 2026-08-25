// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

#[cfg(feature = "link")]
pub mod cgo;

mod dicts;
pub use dicts::Dictionary;

mod types;
pub use types::{BrotliStatus, DEFAULT_WINDOW_SIZE};

#[cfg(feature = "wasmer_traits")]
mod wasmer_traits;

#[cfg(feature = "link")]
mod native;
#[cfg(feature = "link")]
pub use native::*;

#[cfg(not(feature = "link"))]
mod host_imports;
#[cfg(not(feature = "link"))]
pub use host_imports::*;
