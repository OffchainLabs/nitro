//! RPC error types for the arb-rpc handlers.
//!
//! Mirrors reth's `reth-engine-api/src/error.rs` pattern: every variant
//! maps to a JSON-RPC error code via `From<RpcError> for ErrorObjectOwned`.
//! Internal errors are redacted before reaching the wire so we never leak
//! storage / database / consensus details to RPC callers.

use jsonrpsee::types::{
    ErrorObject, ErrorObjectOwned,
    error::{INTERNAL_ERROR_CODE, INTERNAL_ERROR_MSG, INVALID_PARAMS_CODE},
};

use crate::{BlockProducerError, mel::MelProviderError};

/// Result alias for [`RpcError`].
pub type RpcResult<T> = Result<T, RpcError>;

/// Errors surfaced by the Arbitrum RPC handlers.
///
/// Variants split user-input failures (mapped to `-32602 invalid params`)
/// from internal failures (mapped to `-32603 internal error`, with the
/// source redacted before serialization).
#[derive(Debug, thiserror::Error)]
pub enum RpcError {
    /// Failure originating in the wider arbreth error chain.
    #[error(transparent)]
    Arb(#[from] arb_errors::ArbError),

    /// Block production failed downstream of the RPC entrypoint.
    #[error(transparent)]
    BlockProducer(#[from] BlockProducerError),

    /// State-provider lookup failed.
    #[error(transparent)]
    Provider(#[from] reth_storage_errors::provider::ProviderError),

    /// A required RPC parameter was missing or malformed.
    #[error("invalid params: {0}")]
    InvalidParams(String),

    /// A base64-encoded byte field could not be decoded.
    #[error("base64 decode failed: {0}")]
    Base64Decode(String),

    /// A requested resource (block, header) was not found.
    #[error("resource not found: {0}")]
    NotFound(String),

    /// Catch-all for transient internal failures that do not yet have a
    /// dedicated variant. Source is redacted on the wire.
    #[error("internal error: {0}")]
    Internal(String),
}

impl RpcError {
    /// Construct an [`RpcError::InvalidParams`] from a displayable value.
    pub fn invalid_params(msg: impl core::fmt::Display) -> Self {
        Self::InvalidParams(msg.to_string())
    }

    /// Construct an [`RpcError::Base64Decode`] from a displayable value.
    pub fn base64_decode(msg: impl core::fmt::Display) -> Self {
        Self::Base64Decode(msg.to_string())
    }

    /// Construct an [`RpcError::NotFound`] from a displayable value.
    pub fn not_found(msg: impl core::fmt::Display) -> Self {
        Self::NotFound(msg.to_string())
    }

    /// Construct an [`RpcError::Internal`] from a displayable value.
    pub fn internal(msg: impl core::fmt::Display) -> Self {
        Self::Internal(msg.to_string())
    }
}

impl From<MelProviderError> for RpcError {
    fn from(err: MelProviderError) -> Self {
        match err {
            // Nitro sentinels: the message MUST reach the wire verbatim so the
            // client's `strings.Contains(err.Error(), sentinel)` can re-inflate the
            // typed error. Route through an on-wire (non-redacted) variant.
            MelProviderError::AccumulatorMismatch | MelProviderError::FindDelayedNotImplemented => {
                RpcError::InvalidParams(err.to_string())
            }
            MelProviderError::NotFound(msg) => RpcError::NotFound(msg),
            MelProviderError::OutOfBounds { .. } => RpcError::InvalidParams(err.to_string()),
            // Internal backing failures (DB read, L1 fetch, extraction): redacted.
            MelProviderError::Backing(msg) => RpcError::Internal(msg),
        }
    }
}

impl From<MelProviderError> for ErrorObjectOwned {
    fn from(err: MelProviderError) -> Self {
        RpcError::from(err).into()
    }
}

impl From<RpcError> for ErrorObjectOwned {
    fn from(err: RpcError) -> Self {
        match err {
            RpcError::InvalidParams(msg) => {
                ErrorObject::owned(INVALID_PARAMS_CODE, msg, None::<()>)
            }
            RpcError::Base64Decode(msg) => ErrorObject::owned(
                INVALID_PARAMS_CODE,
                format!("base64 decode: {msg}"),
                None::<()>,
            ),
            RpcError::NotFound(msg) => ErrorObject::owned(INVALID_PARAMS_CODE, msg, None::<()>),
            RpcError::Arb(_)
            | RpcError::BlockProducer(_)
            | RpcError::Provider(_)
            | RpcError::Internal(_) => {
                tracing::warn!(target: "arb::rpc", error = %err, "RPC internal error");
                ErrorObject::owned(INTERNAL_ERROR_CODE, INTERNAL_ERROR_MSG, None::<()>)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use jsonrpsee::types::error::{INTERNAL_ERROR_CODE, INVALID_PARAMS_CODE};

    use super::*;

    fn into_obj(err: RpcError) -> ErrorObjectOwned {
        err.into()
    }

    #[test]
    fn invalid_params_maps_to_invalid_params_code() {
        let obj = into_obj(RpcError::invalid_params("bad msg"));
        assert_eq!(obj.code(), INVALID_PARAMS_CODE);
        assert_eq!(obj.message(), "bad msg");
    }

    #[test]
    fn base64_decode_maps_to_invalid_params_code() {
        let obj = into_obj(RpcError::base64_decode("bad char"));
        assert_eq!(obj.code(), INVALID_PARAMS_CODE);
        assert!(obj.message().contains("base64 decode"));
    }

    #[test]
    fn not_found_maps_to_invalid_params_code() {
        let obj = into_obj(RpcError::not_found("block 7"));
        assert_eq!(obj.code(), INVALID_PARAMS_CODE);
        assert_eq!(obj.message(), "block 7");
    }

    #[test]
    fn internal_redacts_source() {
        let obj = into_obj(RpcError::internal("db connection lost"));
        assert_eq!(obj.code(), INTERNAL_ERROR_CODE);
        assert!(!obj.message().contains("db connection lost"));
    }

    #[test]
    fn provider_redacts_source() {
        use reth_storage_errors::provider::ProviderError;
        let obj = into_obj(RpcError::Provider(ProviderError::BestBlockNotFound));
        assert_eq!(obj.code(), INTERNAL_ERROR_CODE);
        assert!(!obj.message().contains("Best block"));
    }

    #[test]
    fn accumulator_mismatch_reaches_wire_verbatim() {
        let obj = into_obj(MelProviderError::AccumulatorMismatch.into());
        assert!(obj.message().contains("delayed message accumulator mismatch"));
    }

    #[test]
    fn find_delayed_not_implemented_reaches_wire_verbatim() {
        let obj = into_obj(MelProviderError::FindDelayedNotImplemented.into());
        assert!(
            obj.message()
                .contains("FindParentChainBlockContainingDelayed is not implemented by MEL")
        );
    }

    #[test]
    fn mel_backing_is_redacted() {
        let obj = into_obj(MelProviderError::Backing("db connection lost".into()).into());
        assert_eq!(obj.code(), INTERNAL_ERROR_CODE);
        assert!(!obj.message().contains("db connection lost"));
    }
}
