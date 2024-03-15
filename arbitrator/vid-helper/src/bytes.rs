//! Convenient serialization for binary blobs.

use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use derive_more::{From, Into};
use serde::{
    de::{Deserialize, Deserializer, Error},
    ser::{Serialize, Serializer},
};
use std::{
    ops::{Deref, DerefMut},
    slice::SliceIndex,
};

/// An unstructured byte array with smart serialization.
///
/// [`Bytes`] mostly acts as a simple byte array, `Vec<u8>`. It can easily be converted to and from
/// a `Vec<u8>`, and it implements many of the same traits as [`Vec`]. In fact internally it merely
/// wraps a `Vec<u8>`.
///
/// The only difference is in how it serializes. `Vec<u8>` serializes very efficiently using
/// `bincode`, but using `serde_json`, it serializes as a JSON array of integers, which is
/// unconventional and inefficient. It is better, in JSON, to serialize binary data as a
/// base64-encoded string. [`Bytes`] uses the [`is_human_readable`](Serializer::is_human_readable)
/// property of a [`Serializer`] to detect whether we are serializing for a compact binary format
/// (like `bincode`) or a human-readable format (like JSON). In the former cases, it serializes
/// directly as an array of bytes. In the latter case, it serializes as a string using base 64.
#[derive(
    Clone,
    Debug,
    Default,
    PartialEq,
    Eq,
    Hash,
    PartialOrd,
    Ord,
    From,
    Into,
    CanonicalSerialize,
    CanonicalDeserialize,
)]
pub struct Bytes(Vec<u8>);

impl Bytes {
    pub fn get<I>(&self, index: I) -> Option<&I::Output>
    where
        I: SliceIndex<[u8]>,
    {
        self.0.get(index)
    }
}

impl Serialize for Bytes {
    fn serialize<S: Serializer>(&self, s: S) -> Result<S::Ok, S::Error> {
        if s.is_human_readable() {
            BASE64.encode(self).serialize(s)
        } else {
            self.0.serialize(s)
        }
    }
}

impl<'a> Deserialize<'a> for Bytes {
    fn deserialize<D: Deserializer<'a>>(d: D) -> Result<Self, D::Error> {
        if d.is_human_readable() {
            Ok(Self(BASE64.decode(String::deserialize(d)?).map_err(
                |err| D::Error::custom(format!("invalid base64: {err}")),
            )?))
        } else {
            Ok(Self(Vec::deserialize(d)?))
        }
    }
}

impl From<&[u8]> for Bytes {
    fn from(bytes: &[u8]) -> Self {
        Self(bytes.into())
    }
}

impl<const N: usize> From<[u8; N]> for Bytes {
    fn from(bytes: [u8; N]) -> Self {
        Self(bytes.into())
    }
}

impl<const N: usize> From<&[u8; N]> for Bytes {
    fn from(bytes: &[u8; N]) -> Self {
        Self((*bytes).into())
    }
}

impl FromIterator<u8> for Bytes {
    fn from_iter<I: IntoIterator<Item = u8>>(iter: I) -> Self {
        Self(iter.into_iter().collect())
    }
}

impl AsRef<[u8]> for Bytes {
    fn as_ref(&self) -> &[u8] {
        self.0.as_ref()
    }
}

impl AsMut<[u8]> for Bytes {
    fn as_mut(&mut self) -> &mut [u8] {
        self.0.as_mut()
    }
}

impl Deref for Bytes {
    type Target = [u8];

    fn deref(&self) -> &[u8] {
        self.as_ref()
    }
}

impl DerefMut for Bytes {
    fn deref_mut(&mut self) -> &mut [u8] {
        self.as_mut()
    }
}

impl PartialEq<[u8]> for Bytes {
    fn eq(&self, other: &[u8]) -> bool {
        self.as_ref() == other
    }
}

impl<const N: usize> PartialEq<[u8; N]> for Bytes {
    fn eq(&self, other: &[u8; N]) -> bool {
        self.as_ref() == other
    }
}

impl PartialEq<Vec<u8>> for Bytes {
    fn eq(&self, other: &Vec<u8>) -> bool {
        self.0 == *other
    }
}

impl<T> Extend<T> for Bytes
where
    Vec<u8>: Extend<T>,
{
    fn extend<I: IntoIterator<Item = T>>(&mut self, iter: I) {
        self.0.extend(iter);
    }
}

#[macro_export]
macro_rules! bytes {
    [$($elem:expr),* $(,)?] => {
        $crate::bytes::Bytes::from(vec![$($elem),*])
    };
    [$elem:expr; $size:expr] => {
        $crate::bytes::Bytes::from(vec![$elem; $size])
    }
}
