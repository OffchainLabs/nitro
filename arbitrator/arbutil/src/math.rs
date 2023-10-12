// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use num_traits::{ops::saturating::SaturatingAdd, Zero};
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

/// Calculates a sum, saturating in cases of overflow.
pub trait SaturatingSum {
    type Number;

    fn saturating_sum(self) -> Self::Number;
}

impl<I, T> SaturatingSum for I
where
    I: Iterator<Item = T>,
    T: SaturatingAdd + Zero,
{
    type Number = T;

    fn saturating_sum(self) -> Self::Number {
        self.fold(T::zero(), |acc, x| acc.saturating_add(&x))
    }
}
