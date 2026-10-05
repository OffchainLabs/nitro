use std::{collections::HashMap, sync::Arc};

use crate::DaReader;

/// Selects the [`DaReader`] for a batch by its header byte.
///
/// The header byte is the first byte of a sequencer message and
/// identifies which DA provider produced the batch.
pub trait DaReaderSource: Send + Sync {
    /// Returns the reader registered for `header_byte`, or `None` if none registered.
    fn get_reader(&self, header_byte: u8) -> Option<Arc<dyn DaReader>>;
}

/// A [`DaReaderSource`] mapping header-byte to [`DaReader`] map.
#[derive(Default)]
pub struct DaReaderRegistry {
    readers: HashMap<u8, Arc<dyn DaReader>>,
}

impl DaReaderRegistry {
    /// Construct an empty registry.
    pub fn new() -> Self {
        Self::default()
    }

    /// Register a reader with the given header byte.
    pub fn register(
        &mut self,
        header_byte: u8,
        reader: Arc<dyn DaReader>,
    ) -> Result<(), DuplicateHeaderByte> {
        use std::collections::hash_map::Entry;

        let entry = self.readers.entry(header_byte);
        match entry {
            Entry::Occupied(_) => Err(DuplicateHeaderByte(header_byte)),
            Entry::Vacant(vacant) => {
                vacant.insert(reader);
                Ok(())
            }
        }
    }
}

impl DaReaderSource for DaReaderRegistry {
    fn get_reader(&self, header_byte: u8) -> Option<Arc<dyn DaReader>> {
        self.readers.get(&header_byte).cloned()
    }
}

/// A reader with the given header byte has already been registered.
#[derive(Debug, thiserror::Error)]
#[error("header byte already in registry: {0:#04x}")]
pub struct DuplicateHeaderByte(u8);

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;

    use super::*;
    use crate::{MockDaReader, Preimages};

    /// The batch coordinates every registered reader is keyed on.
    const BATCH_NUM: u64 = 1;
    const SEQ_MSG: &[u8] = &[0xaa];

    fn block_hash() -> B256 {
        B256::repeat_byte(0x11)
    }

    /// An `Arc<dyn DaReader>` that reads `payload` back for the shared batch
    /// coordinates above.
    fn mock_reader(payload: Vec<u8>) -> Arc<dyn DaReader> {
        let mut mock = MockDaReader::new();
        mock.with_batch(BATCH_NUM, block_hash(), SEQ_MSG, payload, Preimages::new());
        Arc::new(mock)
    }

    /// Reads the payload back through a reader handed out by the registry.
    async fn read(reader: &Arc<dyn DaReader>) -> Vec<u8> {
        reader
            .recover_payload(BATCH_NUM, block_hash(), SEQ_MSG)
            .await
            .unwrap()
    }

    #[tokio::test]
    async fn get_reader_routes_to_registered_reader() {
        let mut registry = DaReaderRegistry::new();
        registry.register(0x80, mock_reader(vec![1, 2, 3])).unwrap();

        let reader = registry.get_reader(0x80).expect("reader was registered");
        assert_eq!(read(&reader).await, vec![1, 2, 3]);
    }

    #[test]
    fn get_reader_unregistered_byte_is_none() {
        let registry = DaReaderRegistry::new();
        assert!(registry.get_reader(0x80).is_none());
    }

    #[test]
    fn get_reader_matches_exact_byte_only() {
        let mut registry = DaReaderRegistry::new();
        registry.register(0x80, mock_reader(vec![0xaa])).unwrap();

        // A registered reader does not answer for any other header byte.
        assert!(registry.get_reader(0x50).is_none());
    }

    #[tokio::test]
    async fn distinct_header_bytes_route_to_distinct_readers() {
        let mut registry = DaReaderRegistry::new();
        registry.register(0x80, mock_reader(vec![0xaa])).unwrap();
        registry.register(0x50, mock_reader(vec![0xbb])).unwrap();

        assert_eq!(read(&registry.get_reader(0x80).unwrap()).await, vec![0xaa]);
        assert_eq!(read(&registry.get_reader(0x50).unwrap()).await, vec![0xbb]);
    }

    #[tokio::test]
    async fn duplicate_registration_errors_and_keeps_original() {
        let mut registry = DaReaderRegistry::new();
        registry.register(0x80, mock_reader(vec![0xaa])).unwrap();

        let err = registry
            .register(0x80, mock_reader(vec![0xbb]))
            .unwrap_err();
        assert_eq!(err.0, 0x80);
        assert_eq!(err.to_string(), "header byte already in registry: 0x80");

        // The failed registration must not overwrite the existing reader.
        assert_eq!(read(&registry.get_reader(0x80).unwrap()).await, vec![0xaa]);
    }
}
