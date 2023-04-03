// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::{
    array::TryFromSliceError,
    borrow::Borrow,
    fmt::{self, Debug, Display, Formatter},
    ops::{Deref, DerefMut},
};

#[derive(Copy, Clone, Default, PartialEq, Eq)]
#[repr(C)]
pub struct Bytes20(pub [u8; 20]);

impl Bytes20 {
    pub fn ptr(&self) -> *const u8 {
        self.0.as_ptr()
    }

    pub fn from_slice(data: &[u8]) -> Result<Self, TryFromSliceError> {
        Ok(Self(data.try_into()?))
    }

    pub fn is_zero(&self) -> bool {
        self == &Bytes20::default()
    }
}

impl Deref for Bytes20 {
    type Target = [u8; 20];

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for Bytes20 {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl AsRef<[u8]> for Bytes20 {
    fn as_ref(&self) -> &[u8] {
        &self.0
    }
}

impl Borrow<[u8]> for Bytes20 {
    fn borrow(&self) -> &[u8] {
        &self.0
    }
}

impl From<[u8; 20]> for Bytes20 {
    fn from(x: [u8; 20]) -> Self {
        Self(x)
    }
}

impl From<u32> for Bytes20 {
    fn from(x: u32) -> Self {
        let mut b = [0u8; 20];
        b[(20 - 4)..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl From<u64> for Bytes20 {
    fn from(x: u64) -> Self {
        let mut b = [0u8; 20];
        b[(20 - 8)..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl From<usize> for Bytes20 {
    fn from(x: usize) -> Self {
        let mut b = [0u8; 20];
        b[(32 - (usize::BITS as usize / 8))..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl Display for Bytes20 {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}

impl Debug for Bytes20 {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}

#[derive(Copy, Clone, Default, PartialEq, Eq)]
#[repr(C)]
pub struct Bytes32(pub [u8; 32]);

impl Bytes32 {
    pub fn ptr(&self) -> *const u8 {
        self.0.as_ptr()
    }

    pub fn from_slice(data: &[u8]) -> Result<Self, TryFromSliceError> {
        Ok(Self(data.try_into()?))
    }

    pub fn is_zero(&self) -> bool {
        self == &Bytes32::default()
    }
}

impl Deref for Bytes32 {
    type Target = [u8; 32];

    fn deref(&self) -> &Self::Target {
        &self.0
    }
}

impl DerefMut for Bytes32 {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.0
    }
}

impl AsRef<[u8]> for Bytes32 {
    fn as_ref(&self) -> &[u8] {
        &self.0
    }
}

impl Borrow<[u8]> for Bytes32 {
    fn borrow(&self) -> &[u8] {
        &self.0
    }
}

impl From<[u8; 32]> for Bytes32 {
    fn from(x: [u8; 32]) -> Self {
        Self(x)
    }
}

impl From<u32> for Bytes32 {
    fn from(x: u32) -> Self {
        let mut b = [0u8; 32];
        b[(32 - 4)..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl From<u64> for Bytes32 {
    fn from(x: u64) -> Self {
        let mut b = [0u8; 32];
        b[(32 - 8)..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl From<usize> for Bytes32 {
    fn from(x: usize) -> Self {
        let mut b = [0u8; 32];
        b[(32 - (usize::BITS as usize / 8))..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl From<Bytes20> for Bytes32 {
    fn from(value: Bytes20) -> Self {
        let mut data = [0; 32];
        data[12..].copy_from_slice(&value.0);
        Self(data)
    }
}

impl Display for Bytes32 {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}

impl Debug for Bytes32 {
    fn fmt(&self, f: &mut Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}
