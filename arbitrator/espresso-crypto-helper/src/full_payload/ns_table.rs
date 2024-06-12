//! Types related to a namespace table.
//!
//! All code that needs to know the binary format of a namespace table is
//! restricted to this file.
//!
//! See [`NsTable`] for a full specification of the binary format of a namespace
//! table.
use crate::{
    full_payload::payload::PayloadByteLen,
    namespace_payload::NsPayloadRange,
    uint_bytes::{bytes_serde_impl, u32_from_bytes, usize_from_bytes, usize_to_bytes},
    NamespaceId,
};
use committable::{Commitment, Committable, RawCommitmentBuilder};
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::{collections::HashSet, sync::Arc};

/// Byte lengths for the different items that could appear in a namespace table.
const NUM_NSS_BYTE_LEN: usize = 4;
const NS_OFFSET_BYTE_LEN: usize = 4;

// TODO prefer [`NS_ID_BYTE_LEN`] set to `8` because [`NamespaceId`] is a `u64`
// but we need to maintain serialization compatibility.
// https://github.com/EspressoSystems/espresso-sequencer/issues/1574
const NS_ID_BYTE_LEN: usize = 4;

/// Raw binary data for a namespace table.
///
/// Any sequence of bytes is a valid [`NsTable`].
///
/// # Binary format of a namespace table
///
/// Byte lengths for the different items that could appear in a namespace table
/// are specified in local private constants [`NUM_NSS_BYTE_LEN`],
/// [`NS_OFFSET_BYTE_LEN`], [`NS_ID_BYTE_LEN`].
///
/// ## Number of entries in the namespace table
///
/// The first [`NUM_NSS_BYTE_LEN`] bytes of the namespace table indicate the
/// number `n` of entries in the table as a little-endian unsigned integer. If
/// the entire table length is smaller than [`NUM_NSS_BYTE_LEN`] then the
/// missing bytes are zero-padded.
///
/// The bytes in the namespace table beyond the first [`NUM_NSS_BYTE_LEN`] bytes
/// encode table entries. Each entry consumes exactly [`NS_ID_BYTE_LEN`] `+`
/// [`NS_OFFSET_BYTE_LEN`] bytes.
///
/// The number `n` could be anything, including a number much larger than the
/// number of entries that could fit in the namespace table. As such, the actual
/// number of entries in the table is defined as the minimum of `n` and the
/// maximum number of whole entries that could fit in the table.
///
/// See [`Self::in_bounds`] for clarification.
///
/// ## Namespace table entry
///
/// ### Namespace ID
///
/// The first [`NS_ID_BYTE_LEN`] bytes of each table entry indicate the
/// [`NamespaceId`] for this namespace. Any table entry whose [`NamespaceId`] is
/// a duplicate of a previous entry is ignored. A correct count of the number of
/// *unique* (non-ignored) entries is given by `NsTable::iter().count()`.
///
/// ### Namespace offset
///
/// The next [`NS_OFFSET_BYTE_LEN`] bytes of each table entry indicate the
/// end-index of a namespace in the block payload bytes
/// [`Payload`](super::payload::Payload). This end-index is a little-endian
/// unsigned integer.
///
/// # How to deduce a namespace's byte range
///
/// In order to extract the payload bytes of a single namespace `N` from the
/// block payload one needs both the start- and end-indices for `N`.
///
/// See [`Self::ns_range`] for clarification. What follows is a description of
/// what's implemented in [`Self::ns_range`].
///
/// If `N` occupies the `i`th entry in the namespace table for `i>0` then the
/// start-index for `N` is defined as the end-index of the `(i-1)`th entry in
/// the table.
///
/// Even if the `(i-1)`the entry would otherwise be ignored (due to a duplicate
/// [`NamespaceId`] or any other reason), that entry's end-index still defines
/// the start-index of `N`. This rule guarantees that both start- and
/// end-indices for any namespace `N` can be read from a constant-size byte
/// range in the namespace table, and it eliminates the need to traverse an
/// unbounded number of previous entries of the namespace table looking for a
/// previous non-ignored entry.
///
/// The start-index of the 0th entry in the table is implicitly defined to be
/// `0`.
///
/// The start- and end-indices `(declared_start, declared_end)` declared in the
/// namespace table could be anything. As such, the actual start- and
/// end-indices `(start, end)` are defined so as to ensure that the byte range
/// is well-defined and in-bounds for the block payload:
/// ```ignore
/// end = min(declared_end, block_payload_byte_length)
/// start = min(declared_start, end)
/// ```
///
/// In a "honestly-prepared" namespace table the end-index of the final
/// namespace equals the byte length of the block payload. (Otherwise the block
/// payload might have bytes that are not included in any namespace.)
///
/// It is possible that a namespace table could indicate two distinct namespaces
/// whose byte ranges overlap, though no "honestly-prepared" namespace table
/// would do this.
///
/// TODO prefer [`NsTable`] to be a newtype like this
/// ```ignore
/// #[repr(transparent)]
/// #[derive(Clone, Debug, Default, Deserialize, Eq, Hash, PartialEq, Serialize)]
/// #[serde(transparent)]
/// pub struct NsTable(#[serde(with = "base64_bytes")] Vec<u8>);
/// ```
/// but we need to maintain serialization compatibility.
/// <https://github.com/EspressoSystems/espresso-sequencer/issues/1575>
#[derive(Clone, Debug, Default, Deserialize, Eq, Hash, PartialEq, Serialize)]
pub struct NsTable {
    #[serde(with = "base64_bytes")]
    bytes: Vec<u8>,
}

impl NsTable {
    /// Search the namespace table for the ns_index belonging to `ns_id`.
    pub fn find_ns_id(&self, ns_id: &NamespaceId) -> Option<NsIndex> {
        self.iter()
            .find(|index| self.read_ns_id_unchecked(index) == *ns_id)
    }

    /// Iterator over all unique namespaces in the namespace table.
    pub fn iter(&self) -> impl Iterator<Item = NsIndex> + '_ {
        NsIter::new(self)
    }

    /// Read the namespace id from the `index`th entry from the namespace table.
    /// Returns `None` if `index` is out of bounds.
    ///
    /// TODO I want to restrict visibility to `pub(crate)` or lower but this
    /// method is currently used in `nasty-client`.
    pub fn read_ns_id(&self, index: &NsIndex) -> Option<NamespaceId> {
        if !self.in_bounds(index) {
            None
        } else {
            Some(self.read_ns_id_unchecked(index))
        }
    }

    /// Like [`Self::read_ns_id`] except `index` is not checked. Use [`Self::in_bounds`] as needed.
    pub fn read_ns_id_unchecked(&self, index: &NsIndex) -> NamespaceId {
        let start = index.0 * (NS_ID_BYTE_LEN + NS_OFFSET_BYTE_LEN) + NUM_NSS_BYTE_LEN;

        // TODO hack to deserialize `NamespaceId` from `NS_ID_BYTE_LEN` bytes
        // https://github.com/EspressoSystems/espresso-sequencer/issues/1574
        NamespaceId::from(u32_from_bytes::<NS_ID_BYTE_LEN>(
            &self.bytes[start..start + NS_ID_BYTE_LEN],
        ))
    }

    /// Does the `index`th entry exist in the namespace table?
    pub fn in_bounds(&self, index: &NsIndex) -> bool {
        // The number of entries in the namespace table, including all duplicate
        // namespace IDs.
        let num_nss_with_duplicates = std::cmp::min(
            // Number of namespaces declared in the ns table
            self.read_num_nss(),
            // Max number of entries that could fit in the namespace table
            self.bytes.len().saturating_sub(NUM_NSS_BYTE_LEN)
                / NS_ID_BYTE_LEN.saturating_add(NS_OFFSET_BYTE_LEN),
        );

        index.0 < num_nss_with_duplicates
    }

    // CRATE-VISIBLE HELPERS START HERE

    /// Read subslice range for the `index`th namespace from the namespace
    /// table.
    pub fn ns_range(&self, index: &NsIndex, payload_byte_len: &PayloadByteLen) -> NsPayloadRange {
        let end = self.read_ns_offset(index).min(payload_byte_len.as_usize());
        let start = if index.0 == 0 {
            0
        } else {
            self.read_ns_offset(&NsIndex(index.0 - 1))
        }
        .min(end);
        NsPayloadRange::new(start, end)
    }

    // PRIVATE HELPERS START HERE

    /// Read the number of namespaces declared in the namespace table. This
    /// quantity might exceed the number of entries that could fit in the
    /// namespace table.
    ///
    /// For a correct count of the number of unique namespaces in this
    /// namespace table use `iter().count()`.
    fn read_num_nss(&self) -> usize {
        let num_nss_byte_len = NUM_NSS_BYTE_LEN.min(self.bytes.len());
        usize_from_bytes::<NUM_NSS_BYTE_LEN>(&self.bytes[..num_nss_byte_len])
    }

    /// Read the namespace offset from the `index`th entry from the namespace table.
    fn read_ns_offset(&self, index: &NsIndex) -> usize {
        let start =
            index.0 * (NS_ID_BYTE_LEN + NS_OFFSET_BYTE_LEN) + NUM_NSS_BYTE_LEN + NS_ID_BYTE_LEN;
        usize_from_bytes::<NS_OFFSET_BYTE_LEN>(&self.bytes[start..start + NS_OFFSET_BYTE_LEN])
    }

    pub fn encode(&self) -> Arc<[u8]> {
        Arc::from(self.bytes.as_ref())
    }
}

impl Committable for NsTable {
    fn commit(&self) -> Commitment<Self> {
        RawCommitmentBuilder::new(&Self::tag())
            .var_size_bytes(&self.bytes)
            .finalize()
    }

    fn tag() -> String {
        "NSTABLE".into()
    }
}

/// Index for an entry in a ns table.
#[derive(Clone, Debug, Eq, Hash, PartialEq)]
pub struct NsIndex(usize);
bytes_serde_impl!(NsIndex, to_bytes, [u8; NUM_NSS_BYTE_LEN], from_bytes);

impl NsIndex {
    pub fn to_bytes(&self) -> [u8; NUM_NSS_BYTE_LEN] {
        usize_to_bytes::<NUM_NSS_BYTE_LEN>(self.0)
    }
    fn from_bytes(bytes: &[u8]) -> Self {
        Self(usize_from_bytes::<NUM_NSS_BYTE_LEN>(bytes))
    }
}

/// Return type for [`Payload::ns_iter`].
pub struct NsIter<'a> {
    cur_index: usize,
    repeat_nss: HashSet<NamespaceId>,
    ns_table: &'a NsTable,
}

impl<'a> NsIter<'a> {
    pub fn new(ns_table: &'a NsTable) -> Self {
        Self {
            cur_index: 0,
            repeat_nss: HashSet::new(),
            ns_table,
        }
    }
}

impl<'a> Iterator for NsIter<'a> {
    type Item = NsIndex;

    fn next(&mut self) -> Option<Self::Item> {
        loop {
            let candidate_result = NsIndex(self.cur_index);
            let ns_id = self.ns_table.read_ns_id(&candidate_result)?;
            self.cur_index += 1;

            // skip duplicate namespace IDs
            if !self.repeat_nss.insert(ns_id) {
                continue;
            }

            break Some(candidate_result);
        }
    }
}
