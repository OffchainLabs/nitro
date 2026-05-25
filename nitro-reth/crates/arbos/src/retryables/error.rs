use arb_storage::StorageError;

use crate::util::BalanceError;

/// Errors raised by the retryable ticket subsystem.
#[derive(Clone, thiserror::Error, Debug)]
pub enum RetryableError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),

    /// `keepalive` was asked to push a ticket's expiry past the maximum
    /// permitted window.
    #[error("timeout too far into the future")]
    TimeoutTooFarFuture,

    /// No retryable ticket exists with the given id.
    #[error("ticket not found")]
    NoTicketWithId,

    /// `cancel` was called by someone other than the ticket's beneficiary.
    #[error("only the beneficiary may cancel a retryable")]
    NotBeneficiary,

    /// A retryable attempted to operate on itself (e.g. redeem from its own
    /// execution context).
    #[error("retryable cannot modify itself")]
    SelfModifyingRetryable,

    /// The transfer callback rejected the escrow movement performed while
    /// processing a retryable.
    #[error(transparent)]
    Balance(#[from] BalanceError),
}
