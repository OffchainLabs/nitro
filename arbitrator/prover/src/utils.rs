// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use eyre::{eyre, Result};
use serde::{Deserialize, Serialize};
use std::{borrow::Borrow, convert::TryInto, fmt, fs::File, io::Read, ops::Deref, path::Path};
use wasmparser::{TableType, Type};

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

pub fn file_bytes(path: &Path) -> Result<Vec<u8>> {
    let mut f = File::open(path)?;
    let mut buf = Vec::new();
    f.read_to_end(&mut buf)?;
    Ok(buf)
}

pub fn split_import(qualified: &str) -> Result<(&str, &str)> {
    let parts: Vec<_> = qualified.split("__").collect();
    let parts = parts.try_into().map_err(|_| eyre!("bad import"))?;
    let [module, name]: [&str; 2] = parts;
    Ok((module, name))
}
