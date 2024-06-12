//! Types related to a namespace payload and its transaction table.
//!
//! All code that needs to know the binary format of a namespace payload and its
//! transaction table is restricted to this file.
//!
//! There are many newtypes in this file to facilitate transaction proofs.
//!
//! # Binary format of a namespace payload
//!
//! Any sequence of bytes is a valid [`NsPayload`].
//!
//! A namespace payload consists of two concatenated byte sequences:
//! - `tx_table`: transaction table
//! - `tx_payloads`: transaction payloads
//!
//! # Transaction table
//!
//! Byte lengths for the different items that could appear in a `tx_table` are
//! specified in local private constants [`NUM_TXS_BYTE_LEN`],
//! [`TX_OFFSET_BYTE_LEN`].
//!
//! ## Number of entries in the transaction table
//!
//! The first [`NUM_TXS_BYTE_LEN`] bytes of the `tx_table` indicate the number
//! `n` of entries in the table as a little-endian unsigned integer. If the
//! entire namespace payload byte length is smaller than [`NUM_TXS_BYTE_LEN`]
//! then the missing bytes are zero-padded.
//!
//! The bytes in the namespace payload beyond the first [`NUM_TXS_BYTE_LEN`]
//! bytes encode entries in the `tx_table`. Each entry consumes exactly
//! [`TX_OFFSET_BYTE_LEN`] bytes.
//!
//! The number `n` could be anything, including a number much larger than the
//! number of entries that could fit in the namespace payload. As such, the
//! actual number of entries in the `tx_table` is defined as the minimum of `n`
//! and the maximum number of whole `tx_table` entries that could fit in the
//! namespace payload.
//!
//! The `tx_payloads` consist of any bytes in the namespace payload beyond the
//! `tx_table`.
//!
//! ## Transaction table entry
//!
//! Each entry in the `tx_table` is exactly [`TX_OFFSET_BYTE_LEN`] bytes. These
//! bytes indicate the end-index of a transaction in the namespace payload
//! bytes. This end-index is a little-endian unsigned integer.
//!
//! This offset is relative to the end of the `tx_table` within the current
//! namespace.
//!
//! ### Example
//!
//! Suppose a block payload has 3000 bytes and 3 namespaces of 1000 bytes each.
//! Suppose the `tx_table` for final namespace in the block has byte length 100,
//! and suppose an entry in that `tx_table` indicates an end-index of `10`. The
//! actual end-index of that transaction relative to the current namespace is
//! `110`: `10` bytes for the offset plus `100` bytes for the `tx_table`.
//! Relative to the entire block payload, the end-index of that transaction is
//! `2110`: `10` bytes for the offset plus `100` bytes for the `tx_table` plus
//! `2000` bytes for this namespace.
//!
//! # How to deduce a transaction's byte range
//!
//! In order to extract the payload bytes of a single transaction `T` from the
//! namespace payload one needs both the start- and end-indices for `T`.
//!
//! See [`TxPayloadRange::new`] for clarification. What follows is a description
//! of what's implemented in [`TxPayloadRange::new`].
//!
//! If `T` occupies the `i`th entry in the `tx_table` for `i>0` then the
//! start-index for `T` is defined as the end-index of the `(i-1)`th entry in
//! the table.
//!
//! Thus, both start- and end-indices for any transaction `T` can be read from a
//! contiguous, constant-size byte range in the `tx_table`. This property
//! facilitates transaction proofs.
//!
//! The start-index of the 0th entry in the table is implicitly defined to be
//! `0`.
//!
//! The start- and end-indices `(declared_start, declared_end)` declared in the
//! `tx_table` could be anything. As such, the actual start- and end-indices
//! `(start, end)` are defined so as to ensure that the byte range is
//! well-defined and in-bounds for the namespace payload:
//! ```ignore
//! end = min(declared_end, namespace_payload_byte_length)
//! start = min(declared_start, end)
//! ```
//!
//! To get the byte range for `T` relative to the current namespace, the above
//! range is translated by the byte length of the `tx_table` *as declared in the
//! `tx_table` itself*, suitably truncated to fit within the current namespace.
//!
//! In particular, if the `tx_table` declares a huge number `n` of entries that
//! cannot fit into the namespace payload then all transactions in this
//! namespace have a zero-length byte range whose start- and end-indices are
//! both `namespace_payload_byte_length`.
//!
//! In a "honestly-prepared" `tx_table` the end-index of the final transaction
//! equals the byte length of the namespace payload minus the byte length of the
//! `tx_table`. (Otherwise the namespace payload might have bytes that are not
//! included in any transaction.)
//!
//! It is possible that a `tx_table` table could indicate two distinct
//! transactions whose byte ranges overlap, though no "honestly-prepared"
//! `tx_table` would do this.
use crate::uint_bytes::{bytes_serde_impl, usize_from_bytes, usize_to_bytes};
use crate::Transaction;
use serde::{Deserialize, Deserializer, Serialize, Serializer};
use std::ops::Range;

/// Byte lengths for the different items that could appear in a tx table.
const NUM_TXS_BYTE_LEN: usize = 4;
const TX_OFFSET_BYTE_LEN: usize = 4;

/// Data that can be deserialized from a subslice of namespace payload bytes.
///
/// Companion trait for [`NsPayloadBytesRange`], which specifies the subslice of
/// namespace payload bytes to read.
pub trait FromNsPayloadBytes<'a> {
    /// Deserialize `Self` from namespace payload bytes.
    fn from_payload_bytes(bytes: &'a [u8]) -> Self;
}

/// Specifies a subslice of namespace payload bytes to read.
///
/// Companion trait for [`FromNsPayloadBytes`], which holds data that can be
/// deserialized from that subslice of bytes.
pub trait NsPayloadBytesRange<'a> {
    type Output: FromNsPayloadBytes<'a>;

    /// Range relative to this ns payload
    fn ns_payload_range(&self) -> Range<usize>;
}

/// Number of txs in a namespace.
///
/// Like [`NumTxsUnchecked`] but checked against a [`NsPayloadByteLen`].
pub struct NumTxs(usize);

impl NumTxs {
    /// Returns the minimum of:
    /// - `num_txs`
    /// - The maximum number of tx table entries that could fit in a namespace
    ///   whose byte length is `byte_len`.
    pub fn new(num_txs: &NumTxsUnchecked, byte_len: &NsPayloadByteLen) -> Self {
        Self(std::cmp::min(
            // Number of txs declared in the tx table
            num_txs.0,
            // Max number of tx table entries that could fit in the namespace payload
            byte_len.0.saturating_sub(NUM_TXS_BYTE_LEN) / TX_OFFSET_BYTE_LEN,
        ))
    }

    pub fn in_bounds(&self, index: &TxIndex) -> bool {
        index.0 < self.0
    }
}

/// Byte length of a namespace payload.
pub struct NsPayloadByteLen(usize);

impl NsPayloadByteLen {
    // TODO restrict visibility?
    pub fn from_usize(n: usize) -> Self {
        Self(n)
    }
}

/// The part of a tx table that declares the number of txs in the payload.
///
/// "Unchecked" because this quantity might exceed the number of tx table
/// entries that could fit into the namespace that contains it.
///
/// Use [`NumTxs`] for the actual number of txs in this namespace.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct NumTxsUnchecked(usize);
bytes_serde_impl!(
    NumTxsUnchecked,
    to_payload_bytes,
    [u8; NUM_TXS_BYTE_LEN],
    from_payload_bytes
);

impl NumTxsUnchecked {
    pub fn to_payload_bytes(&self) -> [u8; NUM_TXS_BYTE_LEN] {
        usize_to_bytes::<NUM_TXS_BYTE_LEN>(self.0)
    }
}

impl FromNsPayloadBytes<'_> for NumTxsUnchecked {
    fn from_payload_bytes(bytes: &[u8]) -> Self {
        Self(usize_from_bytes::<NUM_TXS_BYTE_LEN>(bytes))
    }
}

/// Byte range for the part of a tx table that declares the number of txs in the
/// payload.
pub struct NumTxsRange(Range<usize>);

impl NumTxsRange {
    pub fn new(byte_len: &NsPayloadByteLen) -> Self {
        Self(0..NUM_TXS_BYTE_LEN.min(byte_len.0))
    }
}

impl NsPayloadBytesRange<'_> for NumTxsRange {
    type Output = NumTxsUnchecked;

    fn ns_payload_range(&self) -> Range<usize> {
        self.0.clone()
    }
}

/// Entries from a tx table in a namespace for use in a transaction proof.
///
/// Contains either one or two entries according to whether it was derived from
/// the first transaction in the namespace.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct TxTableEntries {
    cur: usize,
    prev: Option<usize>, // `None` if derived from the first transaction
}

// This serde impl uses Vec. We could save space by using an array of
// length `TWO_ENTRIES_BYTE_LEN`, but then we need a way to distinguish
// `prev=Some(0)` from `prev=None`.
bytes_serde_impl!(
    TxTableEntries,
    to_payload_bytes,
    Vec<u8>,
    from_payload_bytes
);

impl TxTableEntries {
    const TWO_ENTRIES_BYTE_LEN: usize = 2 * TX_OFFSET_BYTE_LEN;

    pub fn to_payload_bytes(&self) -> Vec<u8> {
        let mut bytes = Vec::with_capacity(Self::TWO_ENTRIES_BYTE_LEN);
        if let Some(prev) = self.prev {
            bytes.extend(usize_to_bytes::<TX_OFFSET_BYTE_LEN>(prev));
        }
        bytes.extend(usize_to_bytes::<TX_OFFSET_BYTE_LEN>(self.cur));
        bytes
    }
}

impl FromNsPayloadBytes<'_> for TxTableEntries {
    fn from_payload_bytes(bytes: &[u8]) -> Self {
        match bytes.len() {
            TX_OFFSET_BYTE_LEN => Self {
                cur: usize_from_bytes::<TX_OFFSET_BYTE_LEN>(bytes),
                prev: None,
            },
            Self::TWO_ENTRIES_BYTE_LEN => Self {
                cur: usize_from_bytes::<TX_OFFSET_BYTE_LEN>(&bytes[TX_OFFSET_BYTE_LEN..]),
                prev: Some(usize_from_bytes::<TX_OFFSET_BYTE_LEN>(
                    &bytes[..TX_OFFSET_BYTE_LEN],
                )),
            },
            len => panic!(
                "unexpected bytes len {} should be either {} or {}",
                len,
                TX_OFFSET_BYTE_LEN,
                Self::TWO_ENTRIES_BYTE_LEN
            ),
        }
    }
}

/// Byte range for entries from a tx table for use in a transaction proof.
///
/// This range covers either one or two entries from a tx table according to
/// whether it was derived from the first transaction in the namespace.
pub struct TxTableEntriesRange(Range<usize>);

impl TxTableEntriesRange {
    pub fn new(index: &TxIndex) -> Self {
        let start = if index.0 == 0 {
            // Special case: the desired range includes only one entry from
            // the tx table: the first entry. This entry starts immediately
            // following the bytes that encode the tx table length.
            NUM_TXS_BYTE_LEN
        } else {
            // The desired range starts at the beginning of the previous tx
            // table entry.
            (index.0 - 1)
                .saturating_mul(TX_OFFSET_BYTE_LEN)
                .saturating_add(NUM_TXS_BYTE_LEN)
        };
        // The desired range ends at the end of this transaction's tx table entry
        let end = index
            .0
            .saturating_add(1)
            .saturating_mul(TX_OFFSET_BYTE_LEN)
            .saturating_add(NUM_TXS_BYTE_LEN);
        Self(start..end)
    }
}

impl NsPayloadBytesRange<'_> for TxTableEntriesRange {
    type Output = TxTableEntries;

    fn ns_payload_range(&self) -> Range<usize> {
        self.0.clone()
    }
}

/// A transaction's payload data.
pub struct TxPayload<'a>(&'a [u8]);

impl<'a> TxPayload<'a> {
    pub fn to_payload_bytes(&self) -> &'a [u8] {
        self.0
    }
}

impl<'a> FromNsPayloadBytes<'a> for TxPayload<'a> {
    fn from_payload_bytes(bytes: &'a [u8]) -> Self {
        Self(bytes)
    }
}

/// Byte range for a transaction's payload data.
pub struct TxPayloadRange(Range<usize>);

impl TxPayloadRange {
    pub fn new(
        num_txs: &NumTxsUnchecked,
        tx_table_entries: &TxTableEntries,
        byte_len: &NsPayloadByteLen,
    ) -> Self {
        let tx_table_byte_len = num_txs
            .0
            .saturating_mul(TX_OFFSET_BYTE_LEN)
            .saturating_add(NUM_TXS_BYTE_LEN);
        let end = tx_table_entries
            .cur
            .saturating_add(tx_table_byte_len)
            .min(byte_len.0);
        let start = tx_table_entries
            .prev
            .unwrap_or(0)
            .saturating_add(tx_table_byte_len)
            .min(end);
        Self(start..end)
    }
}

impl<'a> NsPayloadBytesRange<'a> for TxPayloadRange {
    type Output = TxPayload<'a>;

    fn ns_payload_range(&self) -> Range<usize> {
        self.0.clone()
    }
}

/// Index for an entry in a tx table.
#[derive(Clone, Debug, Eq, Hash, PartialEq)]
pub struct TxIndex(usize);
bytes_serde_impl!(TxIndex, to_bytes, [u8; NUM_TXS_BYTE_LEN], from_bytes);

impl TxIndex {
    pub fn to_bytes(&self) -> [u8; NUM_TXS_BYTE_LEN] {
        usize_to_bytes::<NUM_TXS_BYTE_LEN>(self.0)
    }
    fn from_bytes(bytes: &[u8]) -> Self {
        Self(usize_from_bytes::<NUM_TXS_BYTE_LEN>(bytes))
    }
}

pub struct TxIter(Range<usize>);

impl TxIter {
    pub fn new(num_txs: &NumTxs) -> Self {
        Self(0..num_txs.0)
    }
}

// Simple `impl Iterator` delegates to `Range`.
impl Iterator for TxIter {
    type Item = TxIndex;

    fn next(&mut self) -> Option<Self::Item> {
        self.0.next().map(TxIndex)
    }
}

/// Build an individual namespace payload one transaction at a time.
///
/// Use [`Self::append_tx`] to add each transaction. Use [`Self::into_bytes`]
/// when you're done. The returned bytes include a well-formed tx table and all
/// tx payloads.
#[derive(Default)]
pub struct NsPayloadBuilder {
    tx_table_entries: Vec<u8>,
    tx_bodies: Vec<u8>,
}

impl NsPayloadBuilder {
    /// Add a transaction's payload to this namespace
    pub fn append_tx(&mut self, tx: Transaction) {
        self.tx_bodies.extend(tx.into_payload());
        self.tx_table_entries
            .extend(usize_to_bytes::<TX_OFFSET_BYTE_LEN>(self.tx_bodies.len()));
    }

    /// Serialize to bytes and consume self.
    pub fn into_bytes(self) -> Vec<u8> {
        let mut result = Vec::with_capacity(
            NUM_TXS_BYTE_LEN + self.tx_table_entries.len() + self.tx_bodies.len(),
        );
        let num_txs = NumTxsUnchecked(self.tx_table_entries.len() / TX_OFFSET_BYTE_LEN);
        result.extend(num_txs.to_payload_bytes());
        result.extend(self.tx_table_entries);
        result.extend(self.tx_bodies);
        result
    }

    /// Byte length of a namespace with zero transactions.
    ///
    /// Currently this quantity equals the byte length of the tx table header.
    pub const fn fixed_overhead_byte_len() -> usize {
        NUM_TXS_BYTE_LEN
    }

    /// Byte length added to a namespace by a new transaction beyond that
    /// transaction's payload byte length.
    ///
    /// Currently this quantity equals the byte length of a single tx table
    /// entry.
    pub const fn tx_overhead_byte_len() -> usize {
        TX_OFFSET_BYTE_LEN
    }
}
