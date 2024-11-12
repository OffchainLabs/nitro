use super::super::hotshot_types::ViewNumber;
use super::{FeeAccount, FeeAmount};
use crate::NamespaceId;
use committable::{Commitment, Committable};
use ethers_core::types::Signature;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Eq, PartialEq, Deserialize, Serialize, Hash)]
/// Wrapper enum for Full Network Transactions. Each transaction type
/// will be a variant of this enum.
pub enum FullNetworkTx {
    Bid(BidTx),
}

#[derive(Debug, Clone, Eq, PartialEq, Deserialize, Serialize, Hash)]
/// A transaction to bid for the sequencing rights of a namespace. It
/// is the `signed` form of `BidTxBody`. Expected usage is *build*
/// it by calling `signed` on `BidTxBody`.
pub struct BidTx {
    pub(crate) body: BidTxBody,
    pub(crate) signature: Signature,
}

/// A transaction body holding data required for bid submission.
#[derive(Debug, Clone, Eq, PartialEq, Deserialize, Serialize, Hash)]
pub struct BidTxBody {
    /// Account responsible for the signature
    pub(crate) account: FeeAccount,
    /// Fee to be sequenced in the network.  Different than the bid_amount fee
    // FULL_NETWORK_GAS * MINIMUM_GAS_PRICE
    pub(crate) gas_price: FeeAmount,
    /// The bid amount designated in Wei.  This is different than
    /// the sequencing fee (gas price) for this transaction
    pub(crate) bid_amount: FeeAmount,
    /// The URL the HotShot leader will use to request a bundle
    /// from this sequencer if they win the auction
    pub(crate) url: String,
    /// The slot this bid is for
    pub(crate) view: ViewNumber,
    /// The set of namespace ids the sequencer is bidding for
    pub(crate) namespaces: Vec<NamespaceId>,
}

/// The results of an Auction
#[derive(Debug, Clone, Eq, PartialEq, Deserialize, Serialize, Hash)]
pub struct SolverAuctionResults {
    /// view number the results are for
    pub(crate) view_number: ViewNumber,
    /// A list of the bid txs that won
    pub(crate) winning_bids: Vec<BidTx>,
    /// A list of reserve sequencers being used
    pub(crate) reserve_bids: Vec<(NamespaceId, String)>,
}

impl Committable for SolverAuctionResults {
    fn tag() -> String {
        "SOLVER_AUCTION_RESULTS".to_string()
    }

    fn commit(&self) -> Commitment<Self> {
        let comm = committable::RawCommitmentBuilder::new(&Self::tag())
            .fixed_size_field("view_number", &self.view_number.commit().into())
            .array_field(
                "winning_bids",
                &self
                    .winning_bids
                    .iter()
                    .map(Committable::commit)
                    .collect::<Vec<_>>(),
            )
            .array_field(
                "reserve_bids",
                &self
                    .reserve_bids
                    .iter()
                    .map(|(nsid, url)| {
                        // Set a phantom type to make the compiler happy
                        committable::RawCommitmentBuilder::<SolverAuctionResults>::new(
                            "RESERVE_BID",
                        )
                        .u64(nsid.0)
                        .constant_str(url.as_str())
                        .finalize()
                    })
                    .collect::<Vec<_>>(),
            );
        comm.finalize()
    }
}

impl Committable for BidTx {
    fn tag() -> String {
        "BID_TX".to_string()
    }

    fn commit(&self) -> Commitment<Self> {
        let comm = committable::RawCommitmentBuilder::new(&Self::tag())
            .field("body", self.body.commit())
            .fixed_size_field("signature", &self.signature.into());
        comm.finalize()
    }
}

impl Committable for BidTxBody {
    fn tag() -> String {
        "BID_TX_BODY".to_string()
    }

    fn commit(&self) -> Commitment<Self> {
        let comm = committable::RawCommitmentBuilder::new(&Self::tag())
            .fixed_size_field("account", &self.account.to_fixed_bytes())
            .fixed_size_field("gas_price", &self.gas_price.to_fixed_bytes())
            .fixed_size_field("bid_amount", &self.bid_amount.to_fixed_bytes())
            .var_size_field("url", self.url.as_str().as_ref())
            .u64_field("view", self.view.0)
            .array_field(
                "namespaces",
                &self
                    .namespaces
                    .iter()
                    .map(|e| {
                        committable::RawCommitmentBuilder::<BidTxBody>::new("namespace")
                            .u64(e.0)
                            .finalize()
                    })
                    .collect::<Vec<_>>(),
            );
        comm.finalize()
    }
}
