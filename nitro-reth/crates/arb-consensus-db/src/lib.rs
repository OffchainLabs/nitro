//! Typed access to Nitro's consensus database: the classic (non-MEL) `arbitrumdata`
//! schema, layered over a generic [`kv::KvStore`] backend.
//!
//! [`ConsensusDb`] offers typed reads and writes keyed by [`schema`] descriptors. On-disk
//! key and value encodings are byte-compatible with Nitro, so the two can share a schema.

pub mod codecs;
pub mod kv;
pub mod schema;

/// Errors returned by [`ConsensusDb`] operations.
#[derive(Debug, thiserror::Error)]
pub enum ConsensusDbError {
    #[error("rlp decode error: {0}")]
    Rlp(#[from] alloy_rlp::Error),
    #[error(transparent)]
    Store(Box<dyn std::error::Error + Send + Sync + 'static>),

    #[error("invalid stored value")]
    InvalidStoredValue,
    #[error("schema version malformed")]
    MalformedSchemaVersion,
    #[error("schema version mismatch: found {found}, expected {expected}")]
    SchemaVersionMismatch { found: u64, expected: u64 },
}

impl ConsensusDbError {
    /// Wrap a [`kv::KvStore`] error in [`ConsensusDbError::Store`].
    fn from_store(err: impl kv::KvError) -> ConsensusDbError {
        ConsensusDbError::Store(Box::new(err))
    }
}

/// Result alias defaulting the error type to [`ConsensusDbError`].
pub type Result<T, E = ConsensusDbError> = std::result::Result<T, E>;

/// A typed handle to the consensus database, backed by a [`kv::KvStore`].
#[derive(Debug)]
pub struct ConsensusDb<S> {
    store: S,
}

impl<S: kv::KvStore> ConsensusDb<S> {
    /// Open the database over `store`, checking (and migrating) the schema version.
    pub fn open(store: S) -> Result<Self> {
        let mut db = ConsensusDb { store };
        db.check_schema_version()?;
        Ok(db)
    }

    /// Read the typed value stored at `key`, or `None` if absent.
    pub fn get<K: schema::ConsensusDbKey>(&self, key: K) -> Result<Option<K::StoredValue>> {
        self.get_at_key(&schema::key(&key))
    }

    /// Return whether a value is stored at `key`.
    pub fn has<K: schema::ConsensusDbKey>(&self, key: K) -> Result<bool> {
        self.has_at_key(&schema::key(&key))
    }

    /// Write `value` at `key`, overwriting any existing value.
    pub fn put<K: schema::ConsensusDbKey>(&mut self, key: K, value: &K::StoredValue) -> Result<()> {
        self.put_at_key(&schema::key(&key), value)
    }

    /// Delete `key` if present (a no-op otherwise).
    pub fn delete<K: schema::ConsensusDbKey>(&mut self, key: K) -> Result<()> {
        self.delete_at_key(&schema::key(&key))
    }

    /// Delete every entry from `from` (inclusive) to `to` (exclusive).
    pub fn delete_range<K: schema::PositionalKey>(&mut self, from: K, to: K) -> Result<()> {
        self.store
            .delete_range(&schema::key(&from), &schema::key(&to))
            .map_err(ConsensusDbError::from_store)
    }

    /// Apply a batch of typed writes atomically.
    pub fn write_batch(&mut self, batch: ConsensusDbBatch) -> Result<()> {
        self.write_kv_batch(batch.inner)
    }

    /// Iterate all entries under a positional key's prefix, in ascending position order.
    pub fn iter<K: schema::PositionalKey>(
        &self,
    ) -> impl Iterator<Item = Result<(u64, K::StoredValue)>> + '_ {
        self.iter_decoded::<K>(kv::KeyBuf::new())
    }

    /// Iterate entries under a positional key's prefix starting at `from`.
    pub fn iter_from<K: schema::PositionalKey>(
        &self,
        from: K,
    ) -> impl Iterator<Item = Result<(u64, K::StoredValue)>> + '_ {
        let start = schema::ConsensusDbKey::position(&from)
            .expect("positional key has a position")
            .to_be_bytes()
            .to_vec();
        self.iter_decoded::<K>(start)
    }

    /// Iterate a prefix from `start`, decoding each entry into its position and value.
    fn iter_decoded<K: schema::PositionalKey>(
        &self,
        start: kv::KeyBuf,
    ) -> impl Iterator<Item = Result<(u64, K::StoredValue)>> + '_ {
        self.store.iter_prefix(K::PREFIX, start).map(|res| {
            let (k, v) = res.map_err(ConsensusDbError::from_store)?;
            let pos = u64::from_be_bytes(
                k[K::PREFIX.len()..]
                    .try_into()
                    .map_err(|_| ConsensusDbError::InvalidStoredValue)?,
            );
            Ok((pos, schema::ConsensusDbValue::decode(&v)?))
        })
    }

    /// Check stored schema version, and perform migration to current version.
    fn check_schema_version(&mut self) -> Result<()> {
        let mut version = self.get_schema_version()?;
        while version != schema::CURRENT_VERSION {
            let mut batch = kv::Batch::new();
            match version {
                // No-op migration
                0 | 1 => {}
                other => {
                    return Err(ConsensusDbError::SchemaVersionMismatch {
                        found: other,
                        expected: schema::CURRENT_VERSION,
                    });
                }
            }
            version += 1;
            batch.put(schema::DB_SCHEMA_VERSION, encode_schema_version(version));
            self.write_kv_batch(batch)?;
        }
        Ok(())
    }

    /// Read and decode the value at a raw, pre-built key, or `None` if absent.
    pub fn get_at_key<V: schema::ConsensusDbValue>(&self, key: kv::Key) -> Result<Option<V>> {
        let bytes = self.get_raw(key)?;
        bytes.as_deref().map(V::decode).transpose()
    }

    /// Return whether a value is stored at a raw, pre-built key.
    pub fn has_at_key(&self, key: kv::Key) -> Result<bool> {
        self.store.has(key).map_err(ConsensusDbError::from_store)
    }

    /// Encode and write `value` at a raw, pre-built key.
    pub fn put_at_key<V: schema::ConsensusDbValue>(
        &mut self,
        key: kv::Key,
        value: &V,
    ) -> Result<()> {
        self.put_raw(key, value.encode())
    }

    /// Delete a raw, pre-built key if present.
    pub fn delete_at_key(&mut self, key: kv::Key) -> Result<()> {
        self.store.delete(key).map_err(ConsensusDbError::from_store)
    }

    /// Get raw bytes from the key-value store.
    fn get_raw(&self, key: kv::Key) -> Result<Option<kv::Value>> {
        self.store.get(key).map_err(ConsensusDbError::from_store)
    }

    /// Put raw bytes into the key-value store
    fn put_raw(&mut self, key: kv::Key, value: kv::Value) -> Result<()> {
        self.store
            .put(key, value)
            .map_err(ConsensusDbError::from_store)
    }

    /// Write a key-value batch to the store.
    fn write_kv_batch(&mut self, batch: kv::Batch) -> Result<()> {
        self.store
            .write_batch(batch)
            .map_err(ConsensusDbError::from_store)
    }

    /// Get the schema version stored in the key-value store.
    fn get_schema_version(&self) -> Result<u64> {
        self.get_raw(schema::DB_SCHEMA_VERSION)?
            .map_or(Ok(0), decode_schema_version)
    }
}

/// A batch of typed writes, applied atomically via [`ConsensusDb::write_batch`].
#[derive(Debug, Default)]
pub struct ConsensusDbBatch {
    inner: kv::Batch,
}

impl ConsensusDbBatch {
    /// Create an empty batch.
    pub fn new() -> Self {
        Self::default()
    }

    /// Stage a typed write of `value` at `key`.
    pub fn put<K: schema::ConsensusDbKey>(&mut self, key: K, value: &K::StoredValue) {
        self.inner
            .put(schema::key(&key), schema::ConsensusDbValue::encode(value));
    }

    /// Stage a deletion of `key`.
    pub fn delete<K: schema::ConsensusDbKey>(&mut self, key: K) {
        self.inner.delete(schema::key(&key));
    }
}

/// Parse stored value as a raw big-endian u64, not RLP
fn decode_schema_version(bytes: kv::Value) -> Result<u64> {
    Ok(u64::from_be_bytes(
        bytes
            .as_slice()
            .try_into()
            .map_err(|_| ConsensusDbError::MalformedSchemaVersion)?,
    ))
}

/// Encodes schema version as u64 in big-endian format, not RLP.
fn encode_schema_version(version: u64) -> kv::Value {
    version.to_be_bytes().to_vec()
}

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;

    use super::*;
    use crate::{
        kv::{KvStore, MemoryKvStore},
        schema::{
            DB_SCHEMA_VERSION, DelayedMessageCount, MessageCount, MessageResult, MessageResultAt,
        },
    };

    fn with_version(version: u64) -> MemoryKvStore {
        let mut store = MemoryKvStore::new();
        store
            .put(DB_SCHEMA_VERSION, version.to_be_bytes().to_vec())
            .unwrap();
        store
    }

    fn result_at(pos: u64) -> MessageResult {
        MessageResult {
            block_hash: B256::repeat_byte(pos as u8),
            send_root: B256::ZERO,
        }
    }

    #[test]
    fn fresh_open_persists_version_two_as_be8() {
        let db = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        assert_eq!(
            db.store.get(DB_SCHEMA_VERSION).unwrap(),
            Some(2u64.to_be_bytes().to_vec())
        );
    }

    #[test]
    fn open_ratchets_old_versions_to_current() {
        for seed in [0u64, 1] {
            let db = ConsensusDb::open(with_version(seed)).unwrap();
            assert_eq!(
                db.store.get(DB_SCHEMA_VERSION).unwrap(),
                Some(2u64.to_be_bytes().to_vec())
            );
        }
    }

    #[test]
    fn open_current_version_is_noop() {
        let db = ConsensusDb::open(with_version(2)).unwrap();
        assert_eq!(
            db.store.get(DB_SCHEMA_VERSION).unwrap(),
            Some(2u64.to_be_bytes().to_vec())
        );
    }

    #[test]
    fn open_rejects_unknown_version() {
        assert!(matches!(
            ConsensusDb::open(with_version(3)),
            Err(ConsensusDbError::SchemaVersionMismatch {
                found: 3,
                expected: 2
            })
        ));
    }

    #[test]
    fn open_rejects_malformed_version() {
        let mut store = MemoryKvStore::new();
        store.put(DB_SCHEMA_VERSION, vec![1, 2, 3]).unwrap(); // not 8 bytes
        assert!(matches!(
            ConsensusDb::open(store),
            Err(ConsensusDbError::MalformedSchemaVersion)
        ));
    }

    #[test]
    fn typed_put_get_has_delete() {
        let mut db = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        assert_eq!(db.get(MessageCount).unwrap(), None);
        assert!(!db.has(MessageCount).unwrap());

        db.put(MessageCount, &42u64).unwrap();
        assert_eq!(db.get(MessageCount).unwrap(), Some(42));
        assert!(db.has(MessageCount).unwrap());

        db.delete(MessageCount).unwrap();
        assert_eq!(db.get(MessageCount).unwrap(), None);
    }

    #[test]
    fn iter_returns_entries_in_position_order() {
        let mut db = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        for pos in [2u64, 0, 1] {
            db.put(MessageResultAt(pos), &result_at(pos)).unwrap();
        }
        let got: Vec<(u64, B256)> = db
            .iter::<MessageResultAt>()
            .map(|r| {
                let (pos, v) = r.unwrap();
                (pos, v.block_hash)
            })
            .collect();
        assert_eq!(
            got,
            vec![
                (0, B256::repeat_byte(0)),
                (1, B256::repeat_byte(1)),
                (2, B256::repeat_byte(2)),
            ]
        );
    }

    #[test]
    fn iter_from_starts_at_position() {
        let mut db = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        for pos in 0u64..4 {
            db.put(MessageResultAt(pos), &result_at(pos)).unwrap();
        }
        let positions: Vec<u64> = db
            .iter_from::<MessageResultAt>(MessageResultAt(2))
            .map(|r| r.unwrap().0)
            .collect();
        assert_eq!(positions, vec![2, 3]);
    }

    #[test]
    fn batch_applies_writes_in_order() {
        let mut db = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        db.put(MessageCount, &1u64).unwrap();

        let mut batch = ConsensusDbBatch::new();
        batch.put(MessageCount, &7u64);
        batch.put(DelayedMessageCount, &3u64);
        batch.delete(MessageCount); // put then delete the same key within one batch -> gone
        db.write_batch(batch).unwrap();

        assert_eq!(db.get(MessageCount).unwrap(), None);
        assert_eq!(db.get(DelayedMessageCount).unwrap(), Some(3));
    }
}
