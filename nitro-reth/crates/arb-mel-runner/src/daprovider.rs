//! The data-availability provider interface.
//!
//! In nitro the extractor holds a `*daprovider.DAProviderRegistry` and passes it
//! into the extraction function, which uses it to resolve batch payloads by their
//! header byte. The registry itself is not ported yet; this is a small trait so
//! the collaborator exists as an interface (and can be mocked). Real DA reads
//! happen inside the (mocked) extraction step.

use std::collections::HashSet;

/// Resolves data-availability readers for batch header bytes.
pub trait DaProvider: Send + Sync {
    /// Whether a reader is registered for the given batch header byte.
    ///
    /// Placeholder for `DAProviderRegistry.GetReader` until the registry is ported.
    fn has_reader(&self, header_byte: u8) -> bool;
}

/// A [`DaProvider`] for tests, backed by a set of known header bytes.
#[derive(Debug, Default)]
pub struct MockDaProvider {
    readers: HashSet<u8>,
}

impl MockDaProvider {
    /// Creates a provider with no readers.
    pub fn new() -> Self {
        Self::default()
    }

    /// Registers a reader for `header_byte`.
    pub fn with_reader(&mut self, header_byte: u8) -> &mut Self {
        self.readers.insert(header_byte);
        self
    }
}

impl DaProvider for MockDaProvider {
    fn has_reader(&self, header_byte: u8) -> bool {
        self.readers.contains(&header_byte)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn reports_registered_readers() {
        let mut dap = MockDaProvider::new();
        dap.with_reader(0x88);
        assert!(dap.has_reader(0x88));
        assert!(!dap.has_reader(0x99));
    }
}
