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

pub fn slice_with_runoff<T>(data: &[T], start: usize, end: usize) -> &[T] {
    if start >= data.len() || end < start {
        return &[];
    }

    &data[start..end.min(data.len())]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_limit_vec() {
        let testvec = vec![0, 1, 2, 3];
        assert_eq!(slice_with_runoff(&testvec, 4, 4), vec![]);
        assert_eq!(slice_with_runoff(&testvec, 1, 0), vec![]);
        assert_eq!(slice_with_runoff(&testvec, 0, 0), vec![]);
        assert_eq!(slice_with_runoff(&testvec, 0, 1), vec![0]);
        assert_eq!(slice_with_runoff(&testvec, 1, 3), vec![1, 2]);
        assert_eq!(slice_with_runoff(&testvec, 0, 4), vec![0, 1, 2, 3]);
        assert_eq!(slice_with_runoff(&testvec, 0, 5), vec![0, 1, 2, 3]);
        assert_eq!(slice_with_runoff(&testvec, 2, usize::MAX), vec![2, 3]);
    }
}