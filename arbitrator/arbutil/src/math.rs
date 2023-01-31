// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::ops::{BitAnd, Sub};

/// Checks if a number is a power of 2.
pub fn is_power_of_2<T>(value: T) -> bool
where
    T: Sub<Output = T> + BitAnd<Output = T> + PartialOrd<T> + From<u8> + Copy,
{
    if value <= 0.into() {
        return false;
    }
    value & (value - 1.into()) == 0.into()
}
