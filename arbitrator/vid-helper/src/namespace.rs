// TODO import from sequencer: https://github.com/EspressoSystems/nitro-espresso-integration/issues/87
// This module is essentially copy and pasted VID logic from the sequencer repo. It is an unfortunate workaround
// until the VID portion of the sequencer repo is WASM-compatible.
use ark_bls12_381::Bls12_381;
use ark_serialize::{CanonicalDeserialize, CanonicalSerialize};
use core::fmt;
use derivative::Derivative;
use derive_more::{Display, From, Into};
use jf_primitives::vid::{
    payload_prover::{PayloadProver, Statement},
    VidScheme as VidSchemeTrait,
};
use jf_primitives::{
    pcs::{prelude::UnivariateKzgPCS, PolynomialCommitmentScheme},
    vid::advz::payload_prover::LargeRangeProof,
};
use num_traits::PrimInt;
use serde::{Deserialize, Serialize};
use std::default::Default;
use std::mem::size_of;
use std::{marker::PhantomData, ops::Range};
use trait_set::trait_set;

use crate::bytes::Bytes;

trait_set! {
    pub trait TableWordTraits = CanonicalSerialize
        + CanonicalDeserialize
        + TryFrom<usize>
        + TryInto<usize>
        + Default
         + PrimInt
        + std::marker::Sync;

    // Note: this trait is not used yet as for now the Payload structs are only parametrized with the TableWord parameter.
    pub trait OffsetTraits = CanonicalSerialize
        + CanonicalDeserialize
        + TryFrom<usize>
        + TryInto<usize>
        + Default
        + std::marker::Sync;

    // Note: this trait is not used yet as for now the Payload structs are only parametrized with the TableWord parameter.
    pub trait NsIdTraits =CanonicalSerialize + CanonicalDeserialize + Default + std::marker::Sync;
}

#[derive(
    Clone,
    Copy,
    Serialize,
    Deserialize,
    Debug,
    Display,
    PartialEq,
    Eq,
    Hash,
    Into,
    From,
    Default,
    CanonicalDeserialize,
    CanonicalSerialize,
    PartialOrd,
    Ord,
)]
pub struct NamespaceId(pub(crate) u64);

// Use newtype pattern so that tx table entires cannot be confused with other types.
#[derive(Clone, Debug, Deserialize, Eq, Hash, PartialEq, Serialize, Default)]
pub struct TxTableEntry(TxTableEntryWord);
// TODO Get rid of TxTableEntryWord. We might use const generics in order to parametrize the set of functions below with u32,u64  etc...
// See https://github.com/EspressoSystems/espresso-sequencer/issues/1076
pub type TxTableEntryWord = u32;

pub struct TxTable {}
impl TxTable {
    // Parse `TxTableEntry::byte_len()`` bytes from `raw_payload`` starting at `offset` into a `TxTableEntry`
    pub(crate) fn get_len(raw_payload: &[u8], offset: usize) -> TxTableEntry {
        let end = std::cmp::min(
            offset.saturating_add(TxTableEntry::byte_len()),
            raw_payload.len(),
        );
        let start = std::cmp::min(offset, end);
        let tx_table_len_range = start..end;
        let mut entry_bytes = [0u8; TxTableEntry::byte_len()];
        entry_bytes[..tx_table_len_range.len()].copy_from_slice(&raw_payload[tx_table_len_range]);
        TxTableEntry::from_bytes_array(entry_bytes)
    }

    // Parse the table length from the beginning of the tx table inside `ns_bytes`.
    //
    // Returned value is guaranteed to be no larger than the number of tx table entries that could possibly fit into `ns_bytes`.
    // TODO tidy this is a sloppy wrapper for get_len
    pub(crate) fn get_tx_table_len(ns_bytes: &[u8]) -> usize {
        std::cmp::min(
            Self::get_len(ns_bytes, 0).try_into().unwrap_or(0),
            (ns_bytes.len().saturating_sub(TxTableEntry::byte_len())) / TxTableEntry::byte_len(),
        )
    }

    // returns tx_offset
    // if tx_index would reach beyond ns_bytes then return 0.
    // tx_offset is not checked, could be anything
    pub(crate) fn get_table_entry(ns_bytes: &[u8], tx_index: usize) -> usize {
        // get the range for tx_offset bytes in tx table
        let tx_offset_range = {
            let start = std::cmp::min(
                tx_index
                    .saturating_add(1)
                    .saturating_mul(TxTableEntry::byte_len()),
                ns_bytes.len(),
            );
            let end = std::cmp::min(
                start.saturating_add(TxTableEntry::byte_len()),
                ns_bytes.len(),
            );
            start..end
        };

        // parse tx_offset bytes from tx table
        let mut tx_offset_bytes = [0u8; TxTableEntry::byte_len()];
        tx_offset_bytes[..tx_offset_range.len()].copy_from_slice(&ns_bytes[tx_offset_range]);
        usize::try_from(TxTableEntry::from_bytes(&tx_offset_bytes).unwrap_or(TxTableEntry::zero()))
            .unwrap_or(0)
    }
}

impl TxTableEntry {
    pub const MAX: TxTableEntry = Self(TxTableEntryWord::MAX);

    /// Adds `rhs` to `self` in place. Returns `None` on overflow.
    pub fn checked_add_mut(&mut self, rhs: Self) -> Option<()> {
        self.0 = self.0.checked_add(rhs.0)?;
        Some(())
    }
    pub const fn zero() -> Self {
        Self(0)
    }
    pub const fn one() -> Self {
        Self(1)
    }
    pub const fn to_bytes(&self) -> [u8; size_of::<TxTableEntryWord>()] {
        self.0.to_le_bytes()
    }
    pub fn from_bytes(bytes: &[u8]) -> Option<Self> {
        Some(Self(TxTableEntryWord::from_le_bytes(
            bytes.try_into().ok()?,
        )))
    }
    /// Infallible constructor.
    pub fn from_bytes_array(bytes: [u8; TxTableEntry::byte_len()]) -> Self {
        Self(TxTableEntryWord::from_le_bytes(bytes))
    }
    pub const fn byte_len() -> usize {
        size_of::<TxTableEntryWord>()
    }

    pub fn from_usize(val: usize) -> Self {
        Self(
            val.try_into()
                .expect("usize -> TxTableEntry should succeed"),
        )
    }
}

impl fmt::Display for TxTableEntry {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl TryFrom<usize> for TxTableEntry {
    type Error = <TxTableEntryWord as TryFrom<usize>>::Error;

    fn try_from(value: usize) -> Result<Self, Self::Error> {
        TxTableEntryWord::try_from(value).map(Self)
    }
}
impl TryFrom<TxTableEntry> for usize {
    type Error = <usize as TryFrom<TxTableEntryWord>>::Error;

    fn try_from(value: TxTableEntry) -> Result<Self, Self::Error> {
        usize::try_from(value.0)
    }
}

impl TryFrom<NamespaceId> for TxTableEntry {
    type Error = <TxTableEntryWord as TryFrom<u64>>::Error;

    fn try_from(value: NamespaceId) -> Result<Self, Self::Error> {
        TxTableEntryWord::try_from(value.0).map(Self)
    }
}
impl TryFrom<TxTableEntry> for NamespaceId {
    type Error = <u64 as TryFrom<TxTableEntryWord>>::Error;

    fn try_from(value: TxTableEntry) -> Result<Self, Self::Error> {
        Ok(Self(From::from(value.0)))
    }
}

pub type VidScheme = jf_primitives::vid::advz::Advz<ark_bls12_381::Bls12_381, sha2::Sha256>;

/// Namespace proof type
///
/// # Type complexity
///
/// Jellyfish's `LargeRangeProof` type has a prime field generic parameter `F`.
/// This `F` is determined by the pairing parameter for `Advz` currently returned by `test_vid_factory()`.
/// Jellyfish needs a more ergonomic way for downstream users to refer to this type.
///
/// There is a `KzgEval` type alias in jellyfish that helps a little, but it's currently private.
/// If it were public then we could instead use
/// ```compile_fail
/// LargeRangeProof<KzgEval<Bls12_281>>
/// ```
/// but that's still pretty crufty.
pub type JellyfishNamespaceProof =
    LargeRangeProof<<UnivariateKzgPCS<Bls12_381> as PolynomialCommitmentScheme>::Evaluation>;

#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
#[serde(bound = "")] // for V
pub enum NamespaceProof {
    Existence {
        ns_payload_flat: Bytes,
        ns_id: NamespaceId,
        ns_proof: JellyfishNamespaceProof,
        vid_common: <VidScheme as VidSchemeTrait>::Common,
    },
    NonExistence {
        ns_id: NamespaceId,
    },
}

impl NamespaceProof {
    /// Verify a [`NamespaceProof`].
    ///
    /// All args must be available to the verifier in the block header.
    #[allow(dead_code)] // TODO temporary
    pub fn verify(
        &self,
        vid: &VidScheme,
        commit: &<VidScheme as VidSchemeTrait>::Commit,
        ns_table: &NameSpaceTable<TxTableEntryWord>,
    ) -> Option<(Vec<Transaction>, NamespaceId)> {
        match self {
            NamespaceProof::Existence {
                ns_payload_flat,
                ns_id,
                ns_proof,
                vid_common,
            } => {
                let ns_index = ns_table.lookup(*ns_id).unwrap();

                // TODO rework NameSpaceTable struct
                // TODO merge get_ns_payload_range with get_ns_table_entry ?
                let ns_payload_range = ns_table
                    .get_payload_range(ns_index, VidScheme::get_payload_byte_len(vid_common));

                let ns_id = ns_table.get_table_entry(ns_index).0;

                // verify self against args
                vid.payload_verify(
                    Statement {
                        payload_subslice: ns_payload_flat,
                        range: ns_payload_range,
                        commit,
                        common: vid_common,
                    },
                    ns_proof,
                )
                .unwrap()
                .unwrap();

                // verification succeeded, return some data
                // we know ns_id is correct because the corresponding ns_payload_range passed verification
                Some((parse_ns_payload(ns_payload_flat, ns_id), ns_id))
            }
            NamespaceProof::NonExistence { ns_id } => {
                if ns_table.lookup(*ns_id).is_some() {
                    return None; // error: expect not to find ns_id in ns_table
                }
                Some((Vec::new(), *ns_id))
            }
        }
    }
}

pub struct Transaction {
    _namespace: NamespaceId,
    pub payload: Vec<u8>,
}

impl Transaction {
    pub fn new(vm: NamespaceId, payload: Vec<u8>) -> Self {
        Self {
            _namespace: vm,
            payload,
        }
    }
}

// TODO find a home for this function
pub fn parse_ns_payload(ns_payload_flat: &[u8], ns_id: NamespaceId) -> Vec<Transaction> {
    let num_txs = TxTable::get_tx_table_len(ns_payload_flat);
    let tx_bodies_offset = num_txs
        .saturating_add(1)
        .saturating_mul(TxTableEntry::byte_len());
    let mut txs = Vec::with_capacity(num_txs);
    let mut start = tx_bodies_offset;
    for tx_index in 0..num_txs {
        let end = std::cmp::min(
            TxTable::get_table_entry(ns_payload_flat, tx_index).saturating_add(tx_bodies_offset),
            ns_payload_flat.len(),
        );
        let tx_payload_range = Range {
            start: std::cmp::min(start, end),
            end,
        };
        txs.push(Transaction::new(
            ns_id,
            ns_payload_flat[tx_payload_range].to_vec(),
        ));
        start = end;
    }

    txs
}

#[derive(Clone, Debug, Derivative, Deserialize, Eq, Serialize, Default)]
#[derivative(Hash, PartialEq)]
pub struct NameSpaceTable<TableWord: TableWordTraits> {
    pub bytes: Bytes,
    pub phantom: PhantomData<TableWord>,
}

impl<TableWord: TableWordTraits> NameSpaceTable<TableWord> {
    pub fn from_bytes(bytes: impl Into<Bytes>) -> Self {
        Self {
            bytes: bytes.into(),
            phantom: Default::default(),
        }
    }

    /// Find `ns_id` and return its index into this namespace table.
    ///
    /// TODO return Result or Option? Want to avoid catch-all Error type :(
    pub fn lookup(&self, ns_id: NamespaceId) -> Option<usize> {
        // TODO don't use TxTable, need a new method
        let ns_table_len = TxTable::get_tx_table_len(&self.bytes);

        (0..ns_table_len).find(|&ns_index| ns_id == self.get_table_entry(ns_index).0)
    }

    // returns (ns_id, ns_offset)
    // ns_offset is not checked, could be anything
    pub fn get_table_entry(&self, ns_index: usize) -> (NamespaceId, usize) {
        // get the range for ns_id bytes in ns table
        // ensure `range` is within range for ns_table_bytes
        let start = std::cmp::min(
            ns_index
                .saturating_mul(2)
                .saturating_add(1)
                .saturating_mul(TxTableEntry::byte_len()),
            self.bytes.len(),
        );
        let end = std::cmp::min(
            start.saturating_add(TxTableEntry::byte_len()),
            self.bytes.len(),
        );
        let ns_id_range = start..end;

        // parse ns_id bytes from ns table
        // any failure -> VmId(0)
        let mut ns_id_bytes = [0u8; TxTableEntry::byte_len()];
        ns_id_bytes[..ns_id_range.len()].copy_from_slice(&self.bytes[ns_id_range]);
        let ns_id = NamespaceId::try_from(
            TxTableEntry::from_bytes(&ns_id_bytes).unwrap_or(TxTableEntry::zero()),
        )
        .unwrap_or(NamespaceId(0));

        // get the range for ns_offset bytes in ns table
        // ensure `range` is within range for ns_table_bytes
        // TODO refactor range checking code
        let start = end;
        let end = std::cmp::min(
            start.saturating_add(TxTableEntry::byte_len()),
            self.bytes.len(),
        );
        let ns_offset_range = start..end;

        // parse ns_offset bytes from ns table
        // any failure -> 0 offset (?)
        // TODO refactor parsing code?
        let mut ns_offset_bytes = [0u8; TxTableEntry::byte_len()];
        ns_offset_bytes[..ns_offset_range.len()].copy_from_slice(&self.bytes[ns_offset_range]);
        let ns_offset = usize::try_from(
            TxTableEntry::from_bytes(&ns_offset_bytes).unwrap_or(TxTableEntry::zero()),
        )
        .unwrap_or(0);

        (ns_id, ns_offset)
    }

    /// Like `tx_payload_range` except for namespaces.
    /// Returns the byte range for a ns in the block payload bytes.
    ///
    /// Ensures that the returned range is valid: `start <= end <= block_payload_byte_len`.
    pub fn get_payload_range(
        &self,
        ns_index: usize,
        block_payload_byte_len: usize,
    ) -> Range<usize> {
        let end = std::cmp::min(self.get_table_entry(ns_index).1, block_payload_byte_len);
        let start = if ns_index == 0 {
            0
        } else {
            std::cmp::min(self.get_table_entry(ns_index - 1).1, end)
        };
        start..end
    }
}

pub trait Table<TableWord: TableWordTraits> {
    // Read TxTableEntry::byte_len() bytes from `table_bytes` starting at `offset`.
    // if `table_bytes` has too few bytes at this `offset` then pad with zero.
    // Parse these bytes into a `TxTableEntry` and return.
    // Returns raw bytes, no checking for large values
    fn get_table_len(&self, offset: usize) -> TxTableEntry;

    fn byte_len() -> usize {
        size_of::<TableWord>()
    }
}

impl<TableWord: TableWordTraits> Table<TableWord> for NameSpaceTable<TableWord> {
    // TODO (Philippe) avoid code duplication with similar function in TxTable?
    fn get_table_len(&self, offset: usize) -> TxTableEntry {
        let end = std::cmp::min(
            offset.saturating_add(TxTableEntry::byte_len()),
            self.bytes.len(),
        );
        let start = std::cmp::min(offset, end);
        let tx_table_len_range = start..end;
        let mut entry_bytes = [0u8; TxTableEntry::byte_len()];
        entry_bytes[..tx_table_len_range.len()].copy_from_slice(&self.bytes[tx_table_len_range]);
        TxTableEntry::from_bytes_array(entry_bytes)
    }
}
