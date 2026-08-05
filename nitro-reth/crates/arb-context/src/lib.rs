// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//! MOCK (NIT-5205): stand-in for the real `arb-context` crate until the
//! arbitrum-reth sources are migrated. Depends on `arb-storage` to keep a
//! real workspace dependency edge for CI to exercise.

use arb_storage::MockStorage;

/// Per-block execution context standing in for the real `ArbPrecompileCtx`.
#[derive(Debug, Default)]
pub struct MockContext {
    storage: MockStorage,
}

impl MockContext {
    /// Creates a context with empty storage.
    pub fn new() -> Self {
        Self::default()
    }

    /// Gives handlers mutable access to the backing storage.
    pub fn storage_mut(&mut self) -> &mut MockStorage {
        &mut self.storage
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn context_threads_storage() {
        let mut ctx = MockContext::new();
        ctx.storage_mut().put(b"slot".to_vec(), b"1".to_vec());
        assert_eq!(ctx.storage_mut().get(b"slot"), Some(&b"1"[..]));
    }
}
