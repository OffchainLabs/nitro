// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//! MOCK (NIT-5205): stand-in for the real `arb-storage` crate until the
//! arbitrum-reth sources are migrated. Exists so the ported CI (nextest,
//! doctests, clippy, rustdoc, miri) has real code to run against.

/// In-memory key/value store standing in for the real storage backend.
///
/// ```
/// let mut storage = arb_storage::MockStorage::new();
/// storage.put(b"key".to_vec(), b"value".to_vec());
/// assert_eq!(storage.get(b"key"), Some(&b"value"[..]));
/// ```
#[derive(Debug, Default)]
pub struct MockStorage {
    entries: Vec<(Vec<u8>, Vec<u8>)>,
}

impl MockStorage {
    /// Creates an empty store.
    pub fn new() -> Self {
        Self::default()
    }

    /// Inserts or replaces the value stored under `key`.
    pub fn put(&mut self, key: Vec<u8>, value: Vec<u8>) {
        match self.entries.iter_mut().find(|(k, _)| *k == key) {
            Some((_, existing)) => *existing = value,
            None => self.entries.push((key, value)),
        }
    }

    /// Returns the value stored under `key`, if any.
    pub fn get(&self, key: &[u8]) -> Option<&[u8]> {
        self.entries
            .iter()
            .find(|(k, _)| k.as_slice() == key)
            .map(|(_, v)| v.as_slice())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn put_get_roundtrip() {
        let mut storage = MockStorage::new();
        assert_eq!(storage.get(b"key"), None);
        storage.put(b"key".to_vec(), b"value".to_vec());
        assert_eq!(storage.get(b"key"), Some(&b"value"[..]));
        storage.put(b"key".to_vec(), b"other".to_vec());
        assert_eq!(storage.get(b"key"), Some(&b"other"[..]));
    }
}
