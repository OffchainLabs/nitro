use alloy_primitives::{Address, U256};

/// Result alias for transaction-pool operations.
pub type TxPoolResult<T> = Result<T, TxPoolError>;

/// Errors raised when admitting or validating a transaction for the
/// Arbitrum transaction pool.
///
/// Variants are tightly scoped: each represents a single rejection reason
/// that the validator (or pool wrapper) can produce. There is intentionally
/// no `Other(String)` catch-all — extending the surface should be done by
/// adding a new variant.
#[derive(thiserror::Error, Debug)]
#[non_exhaustive]
pub enum TxPoolError {
    /// Transaction type is one of Arbitrum's system-only types
    /// (deposit, internal, retry, submit-retryable, contract, unsigned).
    ///
    /// These are produced by the rollup itself and must never enter the
    /// pool through the public submission path.
    #[error("system transaction type {type_byte:#x} cannot be submitted via the pool")]
    SystemTransactionType {
        /// EIP-2718 type byte that was rejected.
        type_byte: u8,
    },

    /// Blob (EIP-4844) transactions are unsupported on the L2.
    #[error("blob transactions are not supported on this chain")]
    BlobTransactionsDisallowed,

    /// Signer recovery from the transaction signature failed.
    #[error("failed to recover transaction signer")]
    SignerRecoveryFailed,

    /// Encoded transaction exceeds the configured per-tx input cap.
    #[error("oversized data: transaction size {size}, max {max}")]
    OversizedData {
        /// Encoded length of the offending transaction.
        size: usize,
        /// Configured maximum encoded length.
        max: usize,
    },

    /// Gas limit is below the transaction's intrinsic gas requirement.
    #[error("intrinsic gas too low: provided {provided}, required {required}")]
    IntrinsicGasTooLow {
        /// Gas limit supplied by the transaction.
        provided: u64,
        /// Intrinsic gas required for the call's payload.
        required: u64,
    },

    /// Effective tip is below the minimum required by the pool.
    #[error("underpriced: tip {tip}, base fee {base_fee}")]
    Underpriced {
        /// Effective miner tip (priority fee) in wei.
        tip: u128,
        /// Base fee of the next block in wei.
        base_fee: u128,
    },

    /// Transaction nonce is below the sender's on-chain nonce.
    #[error("nonce too low: tx {tx}, sender {sender}")]
    NonceTooLow {
        /// Nonce carried by the transaction.
        tx: u64,
        /// Current on-chain nonce of the sender.
        sender: u64,
    },

    /// Sender cannot cover the up-front cost (value + gas * price) of the
    /// transaction.
    #[error("insufficient funds: required {required}, available {available}")]
    InsufficientFunds {
        /// Up-front cost the transaction must reserve.
        required: U256,
        /// Sender's available balance.
        available: U256,
    },

    /// Transaction's `chain_id` does not match the chain the pool is on.
    #[error("chain id mismatch: tx {tx}, expected {expected}")]
    ChainIdMismatch {
        /// Chain id carried by the transaction.
        tx: u64,
        /// Chain id the pool is configured for.
        expected: u64,
    },

    /// Sender address is present in the ArbOS filtered-transaction set.
    #[error("sender {0} is blocked from submitting transactions")]
    BlockedSender(Address),

    /// Destination address is present in the ArbOS filtered-transaction set.
    #[error("destination {0} is blocked from receiving transactions")]
    BlockedDestination(Address),
}

impl TxPoolError {
    /// Returns `true` when the rejection indicates the transaction itself
    /// is malformed or violates a structural rule (and should not be
    /// re-tried by the caller).
    ///
    /// Mirrors the spirit of reth's `PoolError::is_bad_transaction`.
    pub const fn is_bad_transaction(&self) -> bool {
        match self {
            Self::SystemTransactionType { .. }
            | Self::BlobTransactionsDisallowed
            | Self::SignerRecoveryFailed
            | Self::OversizedData { .. }
            | Self::IntrinsicGasTooLow { .. }
            | Self::ChainIdMismatch { .. }
            | Self::BlockedSender(_)
            | Self::BlockedDestination(_) => true,
            Self::Underpriced { .. }
            | Self::NonceTooLow { .. }
            | Self::InsufficientFunds { .. } => false,
        }
    }
}
