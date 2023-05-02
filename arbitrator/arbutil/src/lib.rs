// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

/// cbindgen:ignore
pub mod color;
pub mod crypto;
pub mod evm;
pub mod format;
pub mod math;
pub mod operator;
pub mod types;

pub use color::{Color, DebugColor};
pub use types::{Bytes20, Bytes32};

#[cfg(feature = "wavm")]
pub mod wavm;

/// Puts an arbitrary type on the heap.
/// Note: the type must be later freed or the value will be leaked.
pub fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}
