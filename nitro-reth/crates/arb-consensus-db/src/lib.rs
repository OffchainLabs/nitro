pub mod kv;
pub mod schema;

#[derive(Debug, thiserror::Error)]
pub enum ConsensusDbError {
    #[error("rlp decode error: {0}")]
    Rlp(#[from] alloy_rlp::Error),
    #[error(transparent)]
    Store(Box<dyn std::error::Error + Send + Sync + 'static>),

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

    pub fn get<T: schema::KeyPrefix + alloy_rlp::Decodable>(&self, pos: u64) -> Result<Option<T>> {
        self.get_rlp(&schema::key(T::PREFIX, pos))
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
    fn put_rlp(&mut self, key: kv::Key, value: impl alloy_rlp::Encodable) -> Result<()> {
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
