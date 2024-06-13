use crate::hotshot_types::{
    vid_scheme, LargeRangeProofType, VidCommitment, VidCommon, VidSchemeType,
};
use crate::{
    full_payload::{NsIndex, NsTable, PayloadByteLen},
    namespace_payload::NsPayloadOwned,
    NamespaceId, Transaction,
};
use jf_vid::{
    payload_prover::{PayloadProver, Statement},
    VidScheme,
};
use serde::{Deserialize, Serialize};

/// Proof of correctness for namespace payload bytes in a block.
#[derive(Clone, Debug, Eq, PartialEq, Serialize, Deserialize)]
pub struct NsProof {
    ns_index: NsIndex,
    ns_payload: NsPayloadOwned,
    ns_proof: Option<LargeRangeProofType>, // `None` if ns_payload is empty
}

impl NsProof {
    /// Verify a [`NsProof`] against a payload commitment. Returns `None` on
    /// error or if verification fails.
    ///
    /// There is no [`NsPayload`](crate::block::namespace_payload::NsPayload)
    /// arg because this data is already included in the [`NsProof`]. See
    /// [`NsProof::new`] for discussion.
    ///
    /// If verification is successful then return `(Vec<Transaction>,
    /// NamespaceId)` obtained by post-processing the underlying
    /// [`NsPayload`](crate::block::namespace_payload::NsPayload). Why? This
    /// method might be run by a client in a WASM environment who might be
    /// running non-Rust code, in which case the client is unable to perform
    /// this post-processing himself.
    pub fn verify(
        &self,
        ns_table: &NsTable,
        commit: &VidCommitment,
        common: &VidCommon,
    ) -> Option<(Vec<Transaction>, NamespaceId)> {
        VidSchemeType::is_consistent(commit, common).ok()?;
        if !ns_table.in_bounds(&self.ns_index) {
            return None; // error: index out of bounds
        }

        let range = ns_table
            .ns_range(&self.ns_index, &PayloadByteLen::from_vid_common(common))
            .as_block_range();

        match (&self.ns_proof, range.is_empty()) {
            (Some(proof), false) => {
                // TODO vid_scheme() arg should be u32 to match get_num_storage_nodes
                // https://github.com/EspressoSystems/HotShot/issues/3298
                let vid = vid_scheme(
                    VidSchemeType::get_num_storage_nodes(common)
                        .try_into()
                        .ok()?, // error: failure to convert u32 to usize
                );

                vid.payload_verify(
                    Statement {
                        payload_subslice: self.ns_payload.as_bytes_slice(),
                        range,
                        commit,
                        common,
                    },
                    proof,
                )
                .ok()? // error: internal to payload_verify()
                .ok()?; // verification failure
            }
            (None, true) => {} // 0-length namespace, nothing to verify
            (None, false) => {
                return None;
            }
            (Some(_), true) => {
                return None;
            }
        }

        // verification succeeded, return some data
        let ns_id = ns_table.read_ns_id_unchecked(&self.ns_index);
        Some((self.ns_payload.export_all_txs(&ns_id), ns_id))
    }
}
