// Copyright 2022-2024, Offchain Labs, Inc.
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
use num_traits::Unsigned;
pub use types::{Bytes20, Bytes32, PreimageType};

/// Puts an arbitrary type on the heap.
/// Note: the type must be later freed or the value will be leaked.
pub fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}

/// Equivalent to &[start..offset], but truncates when out of bounds rather than panicking.
pub fn slice_with_runoff<T, I>(data: &impl AsRef<[T]>, start: I, end: I) -> &[T]
where
    I: TryInto<usize> + Unsigned,
{
    let start = start.try_into().unwrap_or(usize::MAX);
    let end = end.try_into().unwrap_or(usize::MAX);

    let data = data.as_ref();
    if start >= data.len() || end < start {
        return &[];
    }
    &data[start..end.min(data.len())]
}

#[test]
fn test_limit_vec() {
    let testvec = vec![0, 1, 2, 3];
    assert_eq!(slice_with_runoff(&testvec, 4_u32, 4), &testvec[0..0]);
    assert_eq!(slice_with_runoff(&testvec, 1_u16, 0), &testvec[0..0]);
    assert_eq!(slice_with_runoff(&testvec, 0_u64, 0), &testvec[0..0]);
    assert_eq!(slice_with_runoff(&testvec, 0_u32, 1), &testvec[0..1]);
    assert_eq!(slice_with_runoff(&testvec, 1_u64, 3), &testvec[1..3]);
    assert_eq!(slice_with_runoff(&testvec, 0_u16, 4), &testvec[0..4]);
    assert_eq!(slice_with_runoff(&testvec, 0_u8, 5), &testvec[0..4]);
    assert_eq!(slice_with_runoff(&testvec, 2, usize::MAX), &testvec[2..4]);
}
