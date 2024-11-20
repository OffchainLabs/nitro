use super::{
    BlockMerkleCommitment, FeeInfo, FeeMerkleCommitment, L1BlockInfo, ResolvableChainConfig,
    SolverAuctionResults,
};
use crate::hotshot_types::{BuilderCommitment, VidCommitment};
use crate::NsTable;
use ark_serialize::CanonicalSerialize;
use committable::{Commitment, Committable, RawCommitmentBuilder};
use ethers_core::types::Signature as BuilderSignature;
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Deserialize, Serialize, Hash, PartialEq, Eq)]
/// A header is like a [`Block`] with the body replaced by a digest.
pub struct Header {
    /// A commitment to a ChainConfig or a full ChainConfig.
    pub(crate) chain_config: ResolvableChainConfig,
    pub(crate) height: u64,
    pub(crate) timestamp: u64,
    pub(crate) l1_head: u64,
    pub(crate) l1_finalized: Option<L1BlockInfo>,
    pub(crate) payload_commitment: VidCommitment,
    pub(crate) builder_commitment: BuilderCommitment,
    pub(crate) ns_table: NsTable,
    pub(crate) block_merkle_tree_root: BlockMerkleCommitment,
    pub(crate) fee_merkle_tree_root: FeeMerkleCommitment,
    pub(crate) fee_info: Vec<FeeInfo>,
    pub(crate) builder_signature: Vec<BuilderSignature>,
    pub(crate) auction_results: SolverAuctionResults,
}

impl Committable for Header {
    fn commit(&self) -> Commitment<Self> {
        let mut bmt_bytes = vec![];
        self.block_merkle_tree_root
            .serialize_with_mode(&mut bmt_bytes, ark_serialize::Compress::Yes)
            .unwrap();
        let mut fmt_bytes = vec![];
        self.fee_merkle_tree_root
            .serialize_with_mode(&mut fmt_bytes, ark_serialize::Compress::Yes)
            .unwrap();

        RawCommitmentBuilder::new(&Self::tag())
            .field("chain_config", self.chain_config.commit())
            .u64_field("height", self.height)
            .u64_field("timestamp", self.timestamp)
            .u64_field("l1_head", self.l1_head)
            .optional("l1_finalized", &self.l1_finalized)
            .constant_str("payload_commitment")
            .fixed_size_bytes(self.payload_commitment.as_ref().as_ref())
            .constant_str("builder_commitment")
            .fixed_size_bytes(self.builder_commitment.as_ref())
            .field("ns_table", self.ns_table.commit())
            .var_size_field("block_merkle_tree_root", &bmt_bytes)
            .var_size_field("fee_merkle_tree_root", &fmt_bytes)
            .array_field(
                "fee_info",
                &self
                    .fee_info
                    .iter()
                    .map(Committable::commit)
                    .collect::<Vec<_>>(),
            )
            .field("auction_results", self.auction_results.commit())
            .finalize()
    }

    fn tag() -> String {
        "BLOCK".into()
    }
}
