// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#[cfg(feature = "kzg")]
use crate::kzg::ETHEREUM_KZG_SETTINGS;
use arbutil::PreimageType;
#[cfg(feature = "kzg")]
use c_kzg::Blob;
use digest::Digest;
use eyre::{eyre, Result};
use serde::{Deserialize, Serialize};
use sha2::Sha256;
use sha3::Keccak256;
use std::{convert::TryInto, fs::File, io::Read, path::Path};
use wasmparser::{RefType, TableType};

pub use crate::cbytes::CBytes;
#[cfg(feature = "libc")]
pub use crate::cbytes::CBytesIntoIter;

/// Unfortunately, [`wasmparser::RefType`] isn't serde and its contents aren't public.
/// This type enables serde via a 1:1 transmute.
#[derive(Serialize, Deserialize)]
struct RemoteRefType(pub [u8; 4]);

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
    pub initial: u64,
    pub maximum: Option<u64>,
    pub table64: bool,
    pub shared: bool,
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
        #[cfg(feature = "kzg")]
        PreimageType::EthVersionedHash => {
            // TODO: really we should also accept what version it is,
            // but right now only one version is supported by this hash format anyways.
            let blob = Box::new(Blob::from_bytes(preimage)?);
            let commitment = ETHEREUM_KZG_SETTINGS.blob_to_kzg_commitment(&blob)?;
            let mut commitment_hash: [u8; 32] = Sha256::digest(*commitment.to_bytes()).into();
            commitment_hash[0] = 1;
            Ok(commitment_hash)
        }
        #[cfg(not(feature = "kzg"))]
        PreimageType::EthVersionedHash => {
            eyre::bail!("EthVersionedHash preimage hashing requires the 'kzg' feature");
        }
        PreimageType::DACertificate => {
            // There is no way for us to compute the hash of the preimage for DACertificate.
            // For DACertificate, this is only ever called on the flat file initialization path.
            // For now it's okay to return nothing here but if we want to use the flat file
            // initialization path with DACertificate for testing, then we could include
            // the hash in the file too.
            let b = Default::default();
            Ok(b)
        }
    }
}
