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
