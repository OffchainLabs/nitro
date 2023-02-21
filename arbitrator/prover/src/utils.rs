// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use serde::{Deserialize, Serialize};
use std::{
    borrow::Borrow,
    convert::{TryFrom, TryInto},
    fmt,
    fs::File,
    io::Read,
    ops::{Deref, DerefMut},
    path::Path,
};
use wasmparser::{TableType, Type};

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

/// cbindgen:field-names=[bytes]
#[derive(Default, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
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
        b[(32 - 4)..].copy_from_slice(&x.to_be_bytes());
        Self(b)
    }
}

impl From<u64> for Bytes20 {
    fn from(x: u64) -> Self {
        let mut b = [0u8; 20];
        b[(32 - 8)..].copy_from_slice(&x.to_be_bytes());
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

type GenericBytes20 = digest::generic_array::GenericArray<u8, digest::generic_array::typenum::U20>;

impl From<GenericBytes20> for Bytes20 {
    fn from(x: GenericBytes20) -> Self {
        <[u8; 20]>::from(x).into()
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

/// A Vec<u8> allocated with libc::malloc
pub struct CBytes {
    ptr: *mut u8,
    len: usize,
}

impl CBytes {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn as_slice(&self) -> &[u8] {
        unsafe { std::slice::from_raw_parts(self.ptr, self.len) }
    }

    pub unsafe fn from_raw_parts(ptr: *mut u8, len: usize) -> Self {
        Self { ptr, len }
    }
}

impl Default for CBytes {
    fn default() -> Self {
        Self {
            ptr: std::ptr::null_mut(),
            len: 0,
        }
    }
}

impl fmt::Debug for CBytes {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self.as_slice())
    }
}

impl From<&[u8]> for CBytes {
    fn from(slice: &[u8]) -> Self {
        if slice.is_empty() {
            return Self::default();
        }
        unsafe {
            let ptr = libc::malloc(slice.len()) as *mut u8;
            if ptr.is_null() {
                panic!("Failed to allocate memory instantiating CBytes");
            }
            std::ptr::copy_nonoverlapping(slice.as_ptr(), ptr, slice.len());
            Self {
                ptr,
                len: slice.len(),
            }
        }
    }
}

#[derive(Serialize, Deserialize)]
#[serde(remote = "Type")]
enum RemoteType {
    I32,
    I64,
    F32,
    F64,
    V128,
    FuncRef,
    ExternRef,

    // TODO: types removed in wasmparser 0.95+
    ExnRef,
    Func,
    EmptyBlockType,
}

#[derive(Serialize, Deserialize)]
#[serde(remote = "TableType")]
pub struct RemoteTableType {
    #[serde(with = "RemoteType")]
    pub element_type: Type,
    pub initial: u32,
    pub maximum: Option<u32>,
}

impl Drop for CBytes {
    fn drop(&mut self) {
        unsafe { libc::free(self.ptr as _) }
    }
}

impl Clone for CBytes {
    fn clone(&self) -> Self {
        self.as_slice().into()
    }
}

impl Deref for CBytes {
    type Target = [u8];

    fn deref(&self) -> &[u8] {
        self.as_slice()
    }
}

impl AsRef<[u8]> for CBytes {
    fn as_ref(&self) -> &[u8] {
        self.as_slice()
    }
}

impl Borrow<[u8]> for CBytes {
    fn borrow(&self) -> &[u8] {
        self.as_slice()
    }
}

#[derive(Clone)]
pub struct CBytesIntoIter(CBytes, usize);

impl Iterator for CBytesIntoIter {
    type Item = u8;

    fn next(&mut self) -> Option<u8> {
        if self.1 >= self.0.len {
            return None;
        }
        let byte = self.0[self.1];
        self.1 += 1;
        Some(byte)
    }

    fn size_hint(&self) -> (usize, Option<usize>) {
        let len = self.0.len - self.1;
        (len, Some(len))
    }
}

impl IntoIterator for CBytes {
    type Item = u8;
    type IntoIter = CBytesIntoIter;

    fn into_iter(self) -> CBytesIntoIter {
        CBytesIntoIter(self, 0)
    }
}

pub fn file_bytes(path: &Path) -> eyre::Result<Vec<u8>> {
    let mut f = File::open(path)?;
    let mut buf = Vec::new();
    f.read_to_end(&mut buf)?;
    Ok(buf)
}
