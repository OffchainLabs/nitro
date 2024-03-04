// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

/// cbindgen:ignore
pub mod color;
pub mod crypto;
pub mod evm;
pub mod format;
pub mod math;
pub mod operator;
pub mod pricing;
pub mod types;

pub use color::{Color, DebugColor};
pub use types::{Bytes20, Bytes32};

/// Puts an arbitrary type on the heap.
/// Note: the type must be later freed or the value will be leaked.
pub fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}

/// Equivalent to &[start..offset], but truncates when out of bounds rather than panicking.
pub fn slice_with_runoff<T>(data: &impl AsRef<[T]>, start: usize, end: usize) -> &[T] {
    let data = data.as_ref();
    if start >= data.len() || end < start {
        return &[];
    }
    &data[start..end.min(data.len())]
}

#[test]
fn test_limit_vec() {
    let testvec = vec![0, 1, 2, 3];
    assert_eq!(slice_with_runoff(&testvec, 4, 4), &testvec[0..0]);
    assert_eq!(slice_with_runoff(&testvec, 1, 0), &testvec[0..0]);
    assert_eq!(slice_with_runoff(&testvec, 0, 0), &testvec[0..0]);
    assert_eq!(slice_with_runoff(&testvec, 0, 1), &testvec[0..1]);
    assert_eq!(slice_with_runoff(&testvec, 1, 3), &testvec[1..3]);
    assert_eq!(slice_with_runoff(&testvec, 0, 4), &testvec[0..4]);
    assert_eq!(slice_with_runoff(&testvec, 0, 5), &testvec[0..4]);
    assert_eq!(slice_with_runoff(&testvec, 2, usize::MAX), &testvec[2..4]);
}
