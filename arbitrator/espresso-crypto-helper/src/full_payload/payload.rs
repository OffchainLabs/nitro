use crate::full_payload::ns_table::NsTable;
use crate::hotshot_types::{VidCommon, VidSchemeType};
use jf_vid::VidScheme;
use serde::{Deserialize, Serialize};
use std::fmt::Display;

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
