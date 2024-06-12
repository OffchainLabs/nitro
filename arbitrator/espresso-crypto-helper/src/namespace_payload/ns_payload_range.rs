use super::types::{NsPayloadByteLen, NsPayloadBytesRange};
use std::ops::Range;

/// Index range for a namespace payload inside a block payload.
#[derive(Clone, Debug, Eq, Hash, PartialEq)]
pub struct NsPayloadRange(Range<usize>);

impl NsPayloadRange {
    /// TODO restrict visibility?
    pub fn new(start: usize, end: usize) -> Self {
        Self(start..end)
    }

    /// Access the underlying index range for this namespace inside a block
    /// payload.
    pub fn as_block_range(&self) -> Range<usize> {
        self.0.clone()
    }

    /// Return the byte length of this namespace.
    pub fn byte_len(&self) -> NsPayloadByteLen {
        NsPayloadByteLen::from_usize(self.0.len())
    }

    /// Convert a [`NsPayloadBytesRange`] into a range that's relative to the
    /// entire block payload.
    pub fn block_range<'a, R>(&self, range: &R) -> Range<usize>
    where
        R: NsPayloadBytesRange<'a>,
    {
        let range = range.ns_payload_range();
        range.start + self.0.start..range.end + self.0.start
    }
}
