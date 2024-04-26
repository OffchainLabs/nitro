// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use num_enum::{IntoPrimitive, TryFromPrimitive};
use ruint2::Uint;
use serde::{Deserialize, Serialize};
use std::{
    borrow::Borrow,
    fmt,
    ops::{Deref, DerefMut},
};

// These values must be kept in sync with `arbutil/preimage_type.go`,
// and the if statement in `contracts/src/osp/OneStepProverHostIo.sol` (search for "UNKNOWN_PREIMAGE_TYPE").
#[derive(
    Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, TryFromPrimitive, IntoPrimitive,
)]
#[repr(u8)]
pub enum PreimageType {
    Keccak256,
    Sha2_256,
    EthVersionedHash,
}

/// cbindgen:field-names=[bytes]
#[derive(Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
#[repr(C)]
pub struct Bytes32(pub [u8; 32]);

impl Bytes32 {
    pub const fn new(x: [u8; 32]) -> Self {
        Self(x)
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

impl TryFrom<&[u8]> for Bytes32 {
    type Error = std::array::TryFromSliceError;

    fn try_from(value: &[u8]) -> Result<Self, Self::Error> {
        let value: [u8; 32] = value.try_into()?;
        Ok(Self(value))
    }
}

impl TryFrom<Vec<u8>> for Bytes32 {
    type Error = std::array::TryFromSliceError;

    fn try_from(value: Vec<u8>) -> Result<Self, Self::Error> {
        Self::try_from(value.as_slice())
    }
}

impl IntoIterator for Bytes32 {
    type Item = u8;
    type IntoIter = std::array::IntoIter<u8, 32>;

    fn into_iter(self) -> Self::IntoIter {
        IntoIterator::into_iter(self.0)
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

type GenericBytes32 = digest::generic_array::GenericArray<u8, digest::generic_array::typenum::U32>;

impl From<GenericBytes32> for Bytes32 {
    fn from(x: GenericBytes32) -> Self {
        <[u8; 32]>::from(x).into()
    }
}

type U256 = Uint<256, 4>;

impl From<Bytes32> for U256 {
    fn from(value: Bytes32) -> Self {
        U256::from_be_bytes(value.0)
    }
}

impl From<U256> for Bytes32 {
    fn from(value: U256) -> Self {
        Self(value.to_be_bytes())
    }
}

/// cbindgen:field-names=[bytes]
#[derive(Default, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash, Serialize, Deserialize)]
#[repr(C)]
pub struct Bytes20(pub [u8; 20]);

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

impl TryFrom<&[u8]> for Bytes20 {
    type Error = std::array::TryFromSliceError;

    fn try_from(value: &[u8]) -> Result<Self, Self::Error> {
        let value: [u8; 20] = value.try_into()?;
        Ok(Self(value))
    }
}

impl TryFrom<Vec<u8>> for Bytes20 {
    type Error = std::array::TryFromSliceError;

    fn try_from(value: Vec<u8>) -> Result<Self, Self::Error> {
        Self::try_from(value.as_slice())
    }
}

impl IntoIterator for Bytes20 {
    type Item = u8;
    type IntoIter = std::array::IntoIter<u8, 20>;

    fn into_iter(self) -> Self::IntoIter {
        IntoIterator::into_iter(self.0)
    }
}

impl fmt::Display for Bytes20 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}

impl fmt::Debug for Bytes20 {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", hex::encode(self))
    }
}

type GenericBytes20 = digest::generic_array::GenericArray<u8, digest::generic_array::typenum::U20>;

impl From<GenericBytes20> for Bytes20 {
    fn from(x: GenericBytes20) -> Self {
        <[u8; 20]>::from(x).into()
    }
}
