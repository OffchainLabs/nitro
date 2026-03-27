//! Arbitrum primitive types.
//!
//! Transaction types, receipts, gas accounting, and signed transaction
//! handling for Arbitrum's extended EIP-2718 type space.

extern crate alloc;

pub mod arbos_versions;
pub mod error;
pub mod multigas;
pub mod receipt;
pub mod signed_tx;
pub mod tx_types;

pub use receipt::{ArbDepositReceipt, ArbReceipt, ArbReceiptKind, SetArbReceiptFields};
pub use signed_tx::{
    ArbTransactionExt, ArbTransactionSigned, ArbTxTypeLocal, ArbTypedTransaction, RetryTxInfo,
    SubmitRetryableInfo,
};

/// Arbitrum node primitives for use with reth's `NodePrimitives` trait.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ArbPrimitives;

impl reth_primitives_traits::NodePrimitives for ArbPrimitives {
    type Block = alloy_consensus::Block<ArbTransactionSigned, alloy_consensus::Header>;
    type BlockHeader = alloy_consensus::Header;
    type BlockBody = alloy_consensus::BlockBody<ArbTransactionSigned, alloy_consensus::Header>;
    type SignedTx = ArbTransactionSigned;
    type Receipt = ArbReceipt;
}
