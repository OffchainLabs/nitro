// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

pub mod color;
pub mod crypto;
pub mod format;
pub mod math;
pub mod operator;

pub use color::Color;

#[cfg(feature = "wavm")]
pub mod wavm;

/// Puts an arbitrary type on the heap.
/// Note: the type must be later freed or the value will be leaked.
pub fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}
