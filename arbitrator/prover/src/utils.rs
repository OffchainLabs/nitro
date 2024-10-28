// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#[cfg(feature = "native")]
use crate::kzg::ETHEREUM_KZG_SETTINGS;
use arbutil::PreimageType;
#[cfg(feature = "native")]
use c_kzg::{Blob, KzgCommitment};
use digest::Digest;
use eyre::{eyre, Result};
use serde::{Deserialize, Serialize};
use sha2::Sha256;
use sha3::Keccak256;
use std::{borrow::Borrow, convert::TryInto, fmt, fs::File, io::Read, ops::Deref, path::Path};
use wasmparser::{RefType, TableType};

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

// There's no thread safety concerns for CBytes.
// This type is basically a Box<[u8]> (which is Send + Sync) with libc as an allocator.
// Any data races between threads are prevented by Rust borrowing rules,
// and the data isn't thread-local so there's no concern moving it between threads.
unsafe impl Send for CBytes {}
unsafe impl Sync for CBytes {}

/// Unfortunately, [`wasmparser::RefType`] isn't serde and its contents aren't public.
/// This type enables serde via a 1:1 transmute.
#[derive(Serialize, Deserialize)]
struct RemoteRefType(pub [u8; 3]);

impl From<RefType> for RemoteRefType {
    fn from(value: RefType) -> Self {
        unsafe { std::mem::transmute::<RefType, RemoteRefType>(value) }
    }
}

impl From<RemoteRefType> for RefType {
    fn from(value: RemoteRefType) -> Self {
        unsafe { std::mem::transmute::<RemoteRefType, RefType>(value) }
    }
}

mod remote_convert {
    use super::{RefType, RemoteRefType};
    use serde::{Deserialize, Deserializer, Serialize, Serializer};

    pub fn serialize<S: Serializer>(value: &RefType, serializer: S) -> Result<S::Ok, S::Error> {
        RemoteRefType::from(*value).serialize(serializer)
    }

    pub fn deserialize<'de, D: Deserializer<'de>>(deserializer: D) -> Result<RefType, D::Error> {
        Ok(RemoteRefType::deserialize(deserializer)?.into())
    }
}

#[derive(Serialize, Deserialize)]
#[serde(remote = "TableType")]
pub struct RemoteTableType {
    #[serde(with = "remote_convert")]
    pub element_type: RefType,
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

#[cfg(feature = "native")]
pub fn hash_preimage(preimage: &[u8], ty: PreimageType) -> Result<[u8; 32]> {
    match ty {
        PreimageType::Keccak256 => Ok(Keccak256::digest(preimage).into()),
        PreimageType::Sha2_256 => Ok(Sha256::digest(preimage).into()),
        PreimageType::EthVersionedHash => {
            // TODO: really we should also accept what version it is,
            // but right now only one version is supported by this hash format anyways.
            let blob = Box::new(Blob::from_bytes(preimage)?);
            let commitment = KzgCommitment::blob_to_kzg_commitment(&blob, &ETHEREUM_KZG_SETTINGS)?;
            let mut commitment_hash: [u8; 32] = Sha256::digest(&*commitment.to_bytes()).into();
            commitment_hash[0] = 1;
            Ok(commitment_hash)
        }
    }
}
