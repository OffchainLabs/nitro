pub mod codecs;
pub mod kv;
pub mod schema;

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

pub type Result<T, E = ConsensusDbError> = std::result::Result<T, E>;

#[derive(Debug)]
pub struct ConsensusDb<S> {
    store: S,
}

/// A batch of typed writes, applied atomically via [`ConsensusDb::write_batch`].
#[derive(Debug, Default)]
pub struct ConsensusDbBatch {
    inner: kv::Batch,
}

impl ConsensusDbBatch {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn put<K: schema::ConsensusDbKey>(&mut self, key: K, value: &K::StoredValue) {
        self.inner
            .put(schema::key(&key), schema::ConsensusDbValue::encode(value));
    }

    pub fn delete<K: schema::ConsensusDbKey>(&mut self, key: K) {
        self.inner.delete(schema::key(&key));
    }
}

impl<S: kv::KvStore> ConsensusDb<S> {
    pub fn open(store: S) -> Result<Self> {
        let mut db = ConsensusDb { store };
        db.check_schema_version()?;
        Ok(db)
    }

    pub fn get<K: schema::ConsensusDbKey>(&self, key: K) -> Result<Option<K::StoredValue>> {
        self.get_at_key(&schema::key(&key))
    }

    pub fn has<K: schema::ConsensusDbKey>(&self, key: K) -> Result<bool> {
        self.has_at_key(&schema::key(&key))
    }

    pub fn put<K: schema::ConsensusDbKey>(&mut self, key: K, value: &K::StoredValue) -> Result<()> {
        self.put_at_key(&schema::key(&key), value)
    }

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

    pub fn get_at_key<V: schema::ConsensusDbValue>(&self, key: kv::Key) -> Result<Option<V>> {
        let bytes = self.get_raw(key)?;
        bytes.as_deref().map(V::decode).transpose()
    }

    pub fn has_at_key(&self, key: kv::Key) -> Result<bool> {
        self.store.has(key).map_err(ConsensusDbError::from_store)
    }

    pub fn put_at_key<V: schema::ConsensusDbValue>(
        &mut self,
        key: kv::Key,
        value: &V,
    ) -> Result<()> {
        self.put_raw(key, value.encode())
    }

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
