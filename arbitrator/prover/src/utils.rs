use std::{
    borrow::Borrow,
    fmt,
    ops::{Deref, DerefMut},
};
use serde::{Deserialize, Serialize};

/// cbindgen:field-names=[bytes]
#[derive(Default, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[repr(C)]
pub struct Bytes32(pub [u8; 32]);

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

impl IntoIterator for Bytes32 {
    type Item = u8;
    type IntoIter = std::array::IntoIter<u8, 32>;

    fn into_iter(self) -> Self::IntoIter {
        std::array::IntoIter::new(self.0)
    }
}

type GenericBytes32 = digest::generic_array::GenericArray<u8, digest::generic_array::typenum::U32>;

impl From<GenericBytes32> for Bytes32 {
    fn from(x: GenericBytes32) -> Self {
        <[u8; 32]>::from(x).into()
    }
}

impl fmt::Display for Bytes32 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}

impl fmt::Debug for Bytes32 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}
