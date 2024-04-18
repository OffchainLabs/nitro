// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use core::ops::{Add, AddAssign, Deref};

/// Represents a pointer to a Guest WASM's memory.
#[derive(Clone, Copy, Eq, PartialEq)]
#[repr(transparent)]
pub struct GuestPtr(pub u32);

impl Add<u32> for GuestPtr {
    type Output = Self;

    fn add(self, rhs: u32) -> Self::Output {
        Self(self.0 + rhs)
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

impl Deref for GuestPtr {
    type Target = u32;

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl GuestPtr {
    pub fn to_u64(self) -> u64 {
        self.into()
    }
}
