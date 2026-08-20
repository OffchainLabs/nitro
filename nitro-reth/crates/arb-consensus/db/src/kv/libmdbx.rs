//! A disk-backed [`crate::kv::KvStore`] over libmdbx (via reth-libmdbx), using the
//! default (unnamed) table.

use std::{path::Path, time::Duration};

use reth_libmdbx::{Environment, EnvironmentFlags, Geometry, Mode, SyncMode, WriteFlags};

use crate::kv::{Batch, Key, KeyBuf, KvStore, Op, Value};

/// Upper bound of the memory map (libmdbx allocates sparsely and grows on demand).
const MAX_MAP_SIZE: usize = 1 << 40; // 1 TiB
/// Grow the map file in 256 MiB steps.
const GROWTH_STEP: isize = 1 << 28;
/// Reader slots. Must exceed the 256-handle read-txn pool in reth-libmdbx.
const MAX_READERS: u64 = 1024;
/// Under [`SyncMode::SafeNoSync`], flush after this many unsynced bytes...
const SYNC_BYTES: usize = 64 << 20;
/// ...or this long since the last unsteady commit, whichever comes first.
const SYNC_PERIOD: Duration = Duration::from_secs(5);

/// Error from the libmdbx-backed store.
#[derive(Debug, thiserror::Error)]
pub enum LibmdbxOpenError {
    #[error(transparent)]
    Io(#[from] std::io::Error),
    #[error(transparent)]
    Mdbx(#[from] reth_libmdbx::Error),
}

#[derive(Debug)]
pub struct LibmdbxKvStore {
    env: Environment,
}

impl LibmdbxKvStore {
    /// Open (creating if needed) a libmdbx store rooted at directory `path`.
    ///
    /// With `sync_mode` false, commits survive process crash but a machine crash
    /// rolls the whole DB back to the last flushed commit (at most [`SYNC_BYTES`]
    /// or [`SYNC_PERIOD`] behind); graceful shutdown still syncs on close. With
    /// `sync_mode` true, every commit is fsynced before completing.
    pub fn open(path: impl AsRef<Path>, sync_mode: bool) -> Result<Self, LibmdbxOpenError> {
        let path = path.as_ref();
        std::fs::create_dir_all(path)?;
        let sync_mode = if sync_mode {
            SyncMode::Durable
        } else {
            SyncMode::SafeNoSync
        };
        let env = Environment::builder()
            .set_flags(EnvironmentFlags {
                mode: Mode::ReadWrite { sync_mode },
                ..Default::default()
            })
            .set_max_readers(MAX_READERS)
            .set_sync_bytes(SYNC_BYTES)
            .set_sync_period(SYNC_PERIOD)
            .set_geometry(Geometry {
                size: Some(0..MAX_MAP_SIZE),
                growth_step: Some(GROWTH_STEP),
                shrink_threshold: None,
                page_size: None,
            })
            .open(path)?;
        Ok(Self { env })
    }
}

impl KvStore for LibmdbxKvStore {
    type Error = reth_libmdbx::Error;

    fn get(&self, key: Key) -> Result<Option<Value>, Self::Error> {
        let tx = self.env.begin_ro_txn()?;
        let db = tx.open_db(None)?;
        tx.get::<Value>(db.dbi(), key)
    }

    fn has(&self, key: Key) -> Result<bool, Self::Error> {
        Ok(self.get(key)?.is_some())
    }

    fn put(&mut self, key: Key, value: Value) -> Result<(), Self::Error> {
        let tx = self.env.begin_rw_txn()?;
        let db = tx.open_db(None)?;
        tx.put(db.dbi(), key, &value, WriteFlags::UPSERT)?;
        tx.commit()?;
        Ok(())
    }

    fn delete(&mut self, key: Key) -> Result<(), Self::Error> {
        let tx = self.env.begin_rw_txn()?;
        let db = tx.open_db(None)?;
        let _existed = tx.del(db.dbi(), key, None)?;
        tx.commit()?;
        Ok(())
    }

    fn write_batch(&mut self, batch: Batch) -> Result<(), Self::Error> {
        let tx = self.env.begin_rw_txn()?;
        let dbi = tx.open_db(None)?.dbi();
        for op in batch.ops.into_iter() {
            match op {
                Op::Put(k, v) => tx.put(dbi, &k, &v, WriteFlags::UPSERT)?,
                Op::Delete(k) => {
                    tx.del(dbi, &k, None)?;
                }
            }
        }
        tx.commit()?;
        Ok(())
    }

    fn iter_prefix(
        &self,
        prefix: Key,
        start: impl AsRef<[u8]>,
    ) -> Result<Vec<(KeyBuf, Value)>, Self::Error> {
        let tx = self.env.begin_ro_txn()?;
        let mut cur = tx.cursor(tx.open_db(None)?.dbi())?;
        let seek = [prefix, start.as_ref()].concat();
        cur.iter_from::<KeyBuf, Value>(&seek)
            .take_while(|res| res.as_ref().map_or(true, |(k, _)| k.starts_with(prefix)))
            .collect()
    }

    fn delete_range(&mut self, start: Key, end: Key) -> Result<(), Self::Error> {
        let tx = self.env.begin_rw_txn()?;
        let dbi = tx.open_db(None)?.dbi();

        // Collect keys first, to avoid mutating under the cursor
        let keys: Vec<KeyBuf> = tx
            .cursor(dbi)?
            .iter_from::<KeyBuf, Value>(start)
            .take_while(|res| res.as_ref().map_or(true, |(k, _)| k.as_slice() < end))
            .map(|res| res.map(|(k, _)| k))
            .collect::<Result<_, _>>()?;

        for k in keys {
            tx.del(dbi, &k, None)?;
        }
        tx.commit()?;
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use tempfile::TempDir;

    use super::*;
    use crate::kv::{Batch, KvStore};

    /// A fresh disk-backed store in a temp dir. Keep the `TempDir` alive for the
    /// store's lifetime — dropping it removes the directory out from under mdbx.
    fn store() -> (TempDir, LibmdbxKvStore) {
        let dir = TempDir::new().unwrap();
        let s = LibmdbxKvStore::open(dir.path(), false).unwrap();
        (dir, s)
    }

    #[test]
    fn put_get_has_delete() {
        let (_dir, mut s) = store();
        assert_eq!(s.get(b"k").unwrap(), None);
        assert!(!s.has(b"k").unwrap());

        s.put(b"k", vec![1, 2]).unwrap();
        assert_eq!(s.get(b"k").unwrap(), Some(vec![1, 2]));
        assert!(s.has(b"k").unwrap());

        s.put(b"k", vec![3]).unwrap(); // upsert
        assert_eq!(s.get(b"k").unwrap(), Some(vec![3]));

        s.delete(b"k").unwrap();
        assert_eq!(s.get(b"k").unwrap(), None);
        s.delete(b"k").unwrap(); // idempotent
    }

    #[test]
    fn write_batch_applies_in_order() {
        let (_dir, mut s) = store();
        let mut b = Batch::new();
        b.put(b"a".to_vec(), vec![1]);
        b.put(b"b".to_vec(), vec![2]);
        b.delete(b"a".to_vec()); // put then delete the same key in one batch -> gone
        s.write_batch(b).unwrap();
        assert_eq!(s.get(b"a").unwrap(), None);
        assert_eq!(s.get(b"b").unwrap(), Some(vec![2]));
    }

    #[test]
    fn empty_batch_is_noop() {
        let (_dir, mut s) = store();
        s.put(b"k", vec![9]).unwrap();
        s.write_batch(Batch::new()).unwrap();
        assert_eq!(s.get(b"k").unwrap(), Some(vec![9]));
    }

    #[test]
    fn iter_prefix_is_ordered_and_bounded() {
        let (_dir, mut s) = store();
        s.put(b"m3", vec![3]).unwrap();
        s.put(b"m1", vec![1]).unwrap();
        s.put(b"m2", vec![2]).unwrap();
        s.put(b"n1", vec![99]).unwrap();

        let got: Vec<_> = s
            .iter_prefix(b"m", b"")
            .unwrap()
            .into_iter()
            .map(|(_, v)| v)
            .collect();
        assert_eq!(got, vec![vec![1], vec![2], vec![3]]);
    }

    #[test]
    fn iter_prefix_respects_start() {
        let (_dir, mut s) = store();
        s.put(b"m1", vec![1]).unwrap();
        s.put(b"m2", vec![2]).unwrap();
        s.put(b"m3", vec![3]).unwrap();

        let got: Vec<_> = s
            .iter_prefix(b"m", b"2")
            .unwrap()
            .into_iter()
            .map(|(_, v)| v)
            .collect();
        assert_eq!(got, vec![vec![2], vec![3]]);
    }

    #[test]
    fn delete_range_is_half_open() {
        let (_dir, mut s) = store();
        for i in 0u8..5 {
            s.put(&[b'k', i], vec![i]).unwrap();
        }
        s.delete_range(&[b'k', 1], &[b'k', 3]).unwrap(); // [1, 3): removes 1 and 2
        assert!(s.has(&[b'k', 0]).unwrap());
        assert!(!s.has(&[b'k', 1]).unwrap());
        assert!(!s.has(&[b'k', 2]).unwrap());
        assert!(s.has(&[b'k', 3]).unwrap());
        assert!(s.has(&[b'k', 4]).unwrap());
    }

    #[test]
    fn consensus_db_persists_across_reopen() {
        use crate::{ConsensusDb, schema::MessageCount};
        let dir = TempDir::new().unwrap();
        {
            let mut db =
                ConsensusDb::open(LibmdbxKvStore::open(dir.path(), false).unwrap()).unwrap();
            db.put(MessageCount, &7u64).unwrap();
        }
        // Reopen the same directory: the write must have persisted.
        let db = ConsensusDb::open(LibmdbxKvStore::open(dir.path(), false).unwrap()).unwrap();
        assert_eq!(db.get(MessageCount).unwrap(), Some(7));
    }
}
