/// Per-block metadata byte array.
///
/// Format: first byte is version (currently 0), subsequent bytes are
/// bit-packed flags indicating which transactions were timeboosted.
#[derive(Debug, Clone, Default)]
pub struct BlockMetadata(pub Vec<u8>);

impl BlockMetadata {
    pub fn new(data: Vec<u8>) -> Self {
        Self(data)
    }

    /// Returns whether the transaction at `tx_index` was timeboosted.
    ///
    /// Returns `None` if the metadata is empty or too short to cover the index.
    pub fn is_tx_timeboosted(&self, tx_index: usize) -> Option<bool> {
        if self.0.is_empty() {
            return None;
        }
        // First byte is version, remaining bytes are bit flags.
        let byte_index = 1 + tx_index / 8;
        let bit_index = tx_index % 8;
        if byte_index >= self.0.len() {
            return Some(false);
        }
        Some(self.0[byte_index] & (1 << bit_index) != 0)
    }
}
