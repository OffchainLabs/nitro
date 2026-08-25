// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

#[cfg(target_arch = "wasm32")]
use alloc::vec::Vec;
use core::{
    ffi::c_void,
    mem::{self, MaybeUninit},
    ptr,
};

pub mod cgo;
mod dicts;
mod types;

#[cfg(feature = "wasmer_traits")]
mod wasmer_traits;

pub use dicts::Dictionary;
use types::*;
pub use types::{BrotliStatus, DEFAULT_WINDOW_SIZE};

#[cfg(feature = "link")]
mod native;
