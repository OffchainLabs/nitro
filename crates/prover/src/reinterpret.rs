// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use num::Zero;
use std::{
    num::Wrapping,
    ops::{Add, BitAnd, BitOr, BitXor, Div, Mul, Rem, Shl, Shr, Sub},
};

pub trait ReinterpretAsSigned:
    Sized
    + Add<Output = Self>
    + Sub<Output = Self>
    + Mul<Output = Self>
    + Div<Output = Self>
    + Shl<usize, Output = Self>
    + Shr<usize, Output = Self>
    + Rem<Output = Self>
    + BitAnd<Output = Self>
    + BitOr<Output = Self>
    + BitXor<Output = Self>
    + Zero
{
    type Signed: ReinterpretAsUnsigned<Unsigned = Self>;
    fn cast_signed(self) -> Self::Signed;
    fn cast_usize(self) -> usize;

    fn rotl(self, n: usize) -> Self;
    fn rotr(self, n: usize) -> Self;
}

pub trait ReinterpretAsUnsigned:
    Sized
    + Add<Output = Self>
    + Sub<Output = Self>
    + Mul<Output = Self>
    + Div<Output = Self>
    + Shl<usize, Output = Self>
    + Shr<usize, Output = Self>
    + Rem<Output = Self>
    + BitAnd<Output = Self>
    + BitOr<Output = Self>
    + BitXor<Output = Self>
    + Zero
{
    type Unsigned: ReinterpretAsSigned<Signed = Self>;
    fn cast_unsigned(self) -> Self::Unsigned;
}

impl ReinterpretAsSigned for Wrapping<u32> {
    type Signed = Wrapping<i32>;

    fn cast_signed(self) -> Wrapping<i32> {
        Wrapping(self.0 as i32)
    }

    fn cast_usize(self) -> usize {
        self.0 as usize
    }

    fn rotl(self, n: usize) -> Self {
        Wrapping(self.0.rotate_left(n as u32))
    }

    fn rotr(self, n: usize) -> Self {
        Wrapping(self.0.rotate_right(n as u32))
    }
}

impl ReinterpretAsUnsigned for Wrapping<i32> {
    type Unsigned = Wrapping<u32>;

    fn cast_unsigned(self) -> Wrapping<u32> {
        Wrapping(self.0 as u32)
    }
}

impl ReinterpretAsSigned for Wrapping<u64> {
    type Signed = Wrapping<i64>;

    fn cast_signed(self) -> Wrapping<i64> {
        Wrapping(self.0 as i64)
    }

    fn cast_usize(self) -> usize {
        self.0 as usize
    }

    fn rotl(self, n: usize) -> Self {
        Wrapping(self.0.rotate_left(n as u32))
    }

    fn rotr(self, n: usize) -> Self {
        Wrapping(self.0.rotate_right(n as u32))
    }
}

impl ReinterpretAsUnsigned for Wrapping<i64> {
    type Unsigned = Wrapping<u64>;

    fn cast_unsigned(self) -> Wrapping<u64> {
        Wrapping(self.0 as u64)
    }
}
