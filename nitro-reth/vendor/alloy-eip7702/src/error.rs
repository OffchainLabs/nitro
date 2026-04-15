use alloy_primitives::U256;

/// EIP-7702 error.
#[derive(Debug, thiserror::Error)]
pub enum Eip7702Error {
    /// Invalid signature `s` value.
    #[error("invalid signature `s` value: {0}")]
    InvalidSValue(U256),
    /// Signature error.
    #[error(transparent)]
    Signature(#[from] alloy_primitives::SignatureError),
}
