// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use core::ops::{Add, AddAssign};

/// Represents a pointer to a Guest WASM's memory.
#[derive(Clone, Copy, Eq, PartialEq)]
#[repr(transparent)]
pub struct GuestPtr(u32);

impl Add<u32> for GuestPtr {
    type Output = Self;

    fn add(self, rhs: u32) -> Self::Output {
        Self(
            self.0
                .checked_add(rhs)
                .expect("GuestPtr arithmetic overflow"),
        )
    }
}

impl AddAssign<u32> for GuestPtr {
    fn add_assign(&mut self, rhs: u32) {
        *self = *self + rhs;
    }
}

impl From<GuestPtr> for u32 {
    fn from(value: GuestPtr) -> Self {
        value.0
    }
}

impl From<GuestPtr> for u64 {
    fn from(value: GuestPtr) -> Self {
        value.0.into()
    }
}

impl GuestPtr {
    pub const fn new(ptr: u32) -> Self {
        Self(ptr)
    }
}
