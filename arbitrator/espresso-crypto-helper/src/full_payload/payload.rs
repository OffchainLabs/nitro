use crate::hotshot_types::{VidCommon, VidSchemeType};
use crate::{
    full_payload::ns_table::{NsIndex, NsTable},
    namespace_payload::{Index, NsPayload, NsPayloadRange},
    Transaction,
};
use jf_vid::VidScheme;
use serde::{Deserialize, Serialize};
use std::{fmt::Display, sync::Arc};

/// Raw payload data for an entire block.
///
/// A block consists of two sequences of arbitrary bytes:
/// - `ns_table`: namespace table
/// - `ns_payloads`: namespace payloads
///
/// Any sequence of bytes is a valid `ns_table`. Any sequence of bytes is a
/// valid `ns_payloads`. The contents of `ns_table` determine how to interpret
/// `ns_payload`.
///
/// # Namespace table
///
/// See [`NsTable`] for the format of a namespace table.
///
/// # Namespace payloads
///
/// A concatenation of payload bytes for multiple individual namespaces.
/// Namespace boundaries are dictated by `ns_table`. See [`NsPayload`] for the
/// format of a namespace payload.
#[derive(Clone, Debug, Deserialize, Eq, Hash, PartialEq, Serialize)]
pub struct Payload {
    // Concatenated payload bytes for each namespace
    #[serde(with = "base64_bytes")]
    ns_payloads: Vec<u8>,

    ns_table: NsTable,
}

impl Payload {
    pub fn encode(&self) -> Arc<[u8]> {
        Arc::from(self.ns_payloads.as_ref())
    }

    pub fn ns_table(&self) -> &NsTable {
        &self.ns_table
    }

    /// Like [`QueryablePayload::transaction_with_proof`] except without the
    /// proof.
    pub fn transaction(&self, index: &Index) -> Option<Transaction> {
        let ns_id = self.ns_table.read_ns_id(index.ns())?;
        let ns_payload = self.ns_payload(index.ns());
        ns_payload.export_tx(&ns_id, index.tx())
    }

    // CRATE-VISIBLE HELPERS START HERE

    pub fn read_ns_payload(&self, range: &NsPayloadRange) -> &NsPayload {
        NsPayload::from_bytes_slice(&self.ns_payloads[range.as_block_range()])
    }

    /// Convenience wrapper for [`Self::read_ns_payload`].
    ///
    /// `index` is not checked. Use `self.ns_table().in_bounds()` as needed.
    pub fn ns_payload(&self, index: &NsIndex) -> &NsPayload {
        let ns_payload_range = self.ns_table().ns_range(index, &self.byte_len());
        self.read_ns_payload(&ns_payload_range)
    }

    pub fn byte_len(&self) -> PayloadByteLen {
        PayloadByteLen(self.ns_payloads.len())
    }
}

impl Display for Payload {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{self:#?}")
    }
}

/// Byte length of a block payload, which includes all namespaces but *not* the
/// namespace table.
pub struct PayloadByteLen(usize);

impl PayloadByteLen {
    /// Extract payload byte length from a [`VidCommon`] and construct a new [`Self`] from it.
    pub fn from_vid_common(common: &VidCommon) -> Self {
        Self(usize::try_from(VidSchemeType::get_payload_byte_len(common)).unwrap())
    }

    /// Is the payload byte length declared in a [`VidCommon`] equal [`Self`]?
    pub fn is_consistent(&self, common: &VidCommon) -> Result<(), ()> {
        // failure to convert to usize implies that `common` cannot be
        // consistent with `self`.
        let expected =
            usize::try_from(VidSchemeType::get_payload_byte_len(common)).map_err(|_| ())?;

        (self.0 == expected).then_some(()).ok_or(())
    }

    pub fn as_usize(&self) -> usize {
        self.0
    }
}
