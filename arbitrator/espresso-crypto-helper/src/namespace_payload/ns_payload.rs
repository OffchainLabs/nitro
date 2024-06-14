use crate::{
    namespace_payload::types::{
        FromNsPayloadBytes, NsPayloadByteLen, NsPayloadBytesRange, NumTxs, NumTxsRange,
        NumTxsUnchecked, TxIndex, TxIter, TxPayloadRange, TxTableEntriesRange,
    },
    NamespaceId, Transaction,
};
use serde::{Deserialize, Serialize};

/// Raw binary data for a single namespace's payload.
///
/// Any sequence of bytes is a valid [`NsPayload`].
///
/// See module-level documentation [`types`](super::types) for a full
/// specification of the binary format of a namespace.
pub struct NsPayload([u8]);

impl NsPayload {
    pub fn as_bytes_slice(&self) -> &[u8] {
        &self.0
    }
    pub fn byte_len(&self) -> NsPayloadByteLen {
        NsPayloadByteLen::from_usize(self.0.len())
    }

    /// Read and parse bytes from the ns payload.
    ///
    /// Arg `range: &R` is convertible into a `Range<usize>` via
    /// [`NsPayloadBytesRange`]. The payload bytes are parsed into a `R::Output`
    /// via [`FromNsPayloadBytes`].
    pub fn read<'a, R>(&'a self, range: &R) -> R::Output
    where
        R: NsPayloadBytesRange<'a>,
    {
        <R::Output as FromNsPayloadBytes<'a>>::from_payload_bytes(&self.0[range.ns_payload_range()])
    }

    /// Return all transactions in this namespace. The namespace ID for each
    /// returned [`Transaction`] is set to `ns_id`.
    pub fn export_all_txs(&self, ns_id: &NamespaceId) -> Vec<Transaction> {
        let num_txs = self.read_num_txs();
        self.iter_from_num_txs(&num_txs)
            .map(|i| self.tx_from_num_txs(ns_id, &i, &num_txs))
            .collect()
    }

    /// Private helper. (Could be pub if desired.)
    fn read_num_txs(&self) -> NumTxsUnchecked {
        self.read(&NumTxsRange::new(&self.byte_len()))
    }

    /// Private helper
    fn iter_from_num_txs(&self, num_txs: &NumTxsUnchecked) -> TxIter {
        let num_txs = NumTxs::new(num_txs, &self.byte_len());
        TxIter::new(&num_txs)
    }

    /// Private helper
    fn tx_from_num_txs(
        &self,
        ns_id: &NamespaceId,
        index: &TxIndex,
        num_txs_unchecked: &NumTxsUnchecked,
    ) -> Transaction {
        let tx_table_entries = self.read(&TxTableEntriesRange::new(index));
        let tx_range = TxPayloadRange::new(num_txs_unchecked, &tx_table_entries, &self.byte_len());

        // TODO don't copy the tx bytes into the return value
        // https://github.com/EspressoSystems/hotshot-query-service/issues/267
        let tx_payload = self.read(&tx_range).to_payload_bytes().to_vec();
        Transaction::new(*ns_id, tx_payload)
    }
}

#[repr(transparent)]
#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(transparent)]
pub struct NsPayloadOwned(#[serde(with = "base64_bytes")] Vec<u8>);

/// Crazy boilerplate code to make it so that [`NsPayloadOwned`] is to
/// [`NsPayload`] as [`Vec<T>`] is to `[T]`. See [How can I create newtypes for
/// an unsized type and its owned counterpart (like `str` and `String`) in safe
/// Rust? - Stack Overflow](https://stackoverflow.com/q/64977525)
mod ns_payload_owned {
    use super::{NsPayload, NsPayloadOwned};
    use std::borrow::Borrow;
    use std::ops::Deref;

    impl NsPayload {
        // pub(super) because I want it visible everywhere in this file but I
        // also want this boilerplate code quarrantined in `ns_payload_owned`.
        pub(super) fn new_private(p: &[u8]) -> &NsPayload {
            unsafe { &*(p as *const [u8] as *const NsPayload) }
        }
    }

    impl Deref for NsPayloadOwned {
        type Target = NsPayload;
        fn deref(&self) -> &NsPayload {
            NsPayload::new_private(&self.0)
        }
    }

    impl Borrow<NsPayload> for NsPayloadOwned {
        fn borrow(&self) -> &NsPayload {
            self.deref()
        }
    }

    impl ToOwned for NsPayload {
        type Owned = NsPayloadOwned;
        fn to_owned(&self) -> NsPayloadOwned {
            NsPayloadOwned(self.0.to_owned())
        }
    }
}
