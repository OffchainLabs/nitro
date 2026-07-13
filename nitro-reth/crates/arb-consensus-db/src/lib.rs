pub mod kv;
pub mod rlp;
pub mod schema;

use schema::ConsensusDbValue;

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

impl<S: kv::KvStore> ConsensusDb<S> {
    pub fn open(store: S) -> Result<Self> {
        let mut db = ConsensusDb { store };
        db.check_schema_version()?;
        Ok(db)
    }

    pub fn get<K: schema::ConsensusDbKey>(&self, key: K) -> Result<Option<K::StoredValue>> {
        self.get_at_key(&key.key())
    }

    pub fn put<K: schema::ConsensusDbKey>(&mut self, key: K, value: &K::StoredValue) -> Result<()> {
        self.put_raw(&key.key(), value.encode())
    }

    pub fn message_count(&self) -> Result<Option<u64>> {
        self.get_rlp(schema::MESSAGE_COUNT_KEY)
    }

    pub fn set_message_count(&mut self, n: u64) -> Result<()> {
        self.put_rlp(schema::MESSAGE_COUNT_KEY, n)
    }

    pub fn delayed_message_count(&self) -> Result<Option<u64>> {
        self.get_rlp(schema::DELAYED_MESSAGE_COUNT_KEY)
    }

    pub fn set_delayed_message_count(&mut self, n: u64) -> Result<()> {
        self.put_rlp(schema::DELAYED_MESSAGE_COUNT_KEY, n)
    }

    pub fn sequencer_batch_count(&self) -> Result<Option<u64>> {
        self.get_rlp(schema::SEQUENCER_BATCH_COUNT_KEY)
    }

    pub fn set_sequencer_batch_count(&mut self, n: u64) -> Result<()> {
        self.put_rlp(schema::SEQUENCER_BATCH_COUNT_KEY, n)
    }

    pub fn last_pruned_message(&self) -> Result<u64> {
        Ok(self.get_rlp(schema::LAST_PRUNED_MESSAGE_KEY)?.unwrap_or(0))
    }

    pub fn set_last_pruned_message(&mut self, n: u64) -> Result<()> {
        self.put_rlp(schema::LAST_PRUNED_MESSAGE_KEY, n)
    }

    pub fn last_pruned_delayed_message(&self) -> Result<u64> {
        Ok(self
            .get_rlp(schema::LAST_PRUNED_DELAYED_MESSAGE_KEY)?
            .unwrap_or(0))
    }

    pub fn set_last_pruned_delayed_message(&mut self, n: u64) -> Result<()> {
        self.put_rlp(schema::LAST_PRUNED_DELAYED_MESSAGE_KEY, n)
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

    pub fn put_at_key<V: schema::ConsensusDbValue>(&mut self, key: kv::Key, value: &V) -> Result<()> {
        self.put_raw(key, value.encode())
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

    /// RLP-decode the value at `key`. Returns `Ok(None)` if the key is absent.
    pub fn get_rlp<T: alloy_rlp::Decodable>(&self, key: kv::Key) -> Result<Option<T>> {
        self.get_raw(key)?
            .map(alloy_rlp::decode_exact)
            .transpose()
            .map_err(ConsensusDbError::Rlp)
    }

    /// RLP-encode `value` and store it at `key` (upsert).
    pub fn put_rlp(&mut self, key: kv::Key, value: impl alloy_rlp::Encodable) -> Result<()> {
        self.put_raw(key, alloy_rlp::encode(value))
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
