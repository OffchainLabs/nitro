use alloy_primitives::Bytes;
use arb_storage_errors::StorageError;
use core::error::Error;
use revm::precompile::{PrecompileError, PrecompileOutput, PrecompileResult};
use std::borrow::Cow;

/// Errors raised by Arbitrum precompiles.
///
/// `Revert` and `OutOfGas` are user-visible: the transaction reverts and the
/// surrounding block continues. `Fatal` indicates an infrastructure failure
/// (database error, broken storage invariant) and must abort the block.
#[derive(thiserror::Error, Debug)]
pub enum ArbPrecompileError {
    /// User-visible revert. The block continues; the tx reverts.
    #[error("revert: {data:?}")]
    Revert {
        /// Optional Solidity custom-error selector that prefixes `data`.
        selector: Option<[u8; 4]>,
        /// ABI-encoded revert payload returned to the caller.
        data: Bytes,
        /// Precompile gas accounted up to the point of the revert.
        gas_used: u64,
    },

    /// Out of gas during precompile execution. User-visible.
    #[error("out of gas")]
    OutOfGas,

    /// Infrastructure failure. Maps to `PrecompileError::Fatal` so the block
    /// aborts instead of producing a user-visible revert.
    #[error("fatal precompile error: {0}")]
    Fatal(#[source] Box<dyn Error + Send + Sync>),
}

impl ArbPrecompileError {
    /// Wraps any [`Error`] as a [`ArbPrecompileError::Fatal`].
    pub fn fatal<E>(err: E) -> Self
    where
        E: Error + Send + Sync + 'static,
    {
        Self::Fatal(Box::new(err))
    }

    /// Builds a [`Revert`](Self::Revert) carrying empty data and the
    /// current precompile-gas accumulator.
    pub fn empty_revert(gas_used: u64) -> Self {
        Self::Revert {
            selector: None,
            data: Bytes::new(),
            gas_used,
        }
    }

    /// Converts this error into a [`PrecompileResult`], capped by `gas_limit`.
    ///
    /// `Revert` produces a successful `PrecompileOutput::new_reverted` carrying
    /// the configured selector and payload. `OutOfGas` and `Fatal` become
    /// `Err`-variant `PrecompileError`s.
    pub fn into_precompile_result(self, gas_limit: u64) -> PrecompileResult {
        match self {
            Self::Revert {
                selector,
                data,
                gas_used,
            } => {
                let payload = match selector {
                    Some(sel) => {
                        let mut bytes = Vec::with_capacity(4 + data.len());
                        bytes.extend_from_slice(&sel);
                        bytes.extend_from_slice(&data);
                        Bytes::from(bytes)
                    }
                    None => data,
                };
                Ok(PrecompileOutput::new_reverted(
                    gas_used.min(gas_limit),
                    payload,
                ))
            }
            other => Err(other.into()),
        }
    }
}

impl From<StorageError> for ArbPrecompileError {
    fn from(err: StorageError) -> Self {
        Self::Fatal(Box::new(err))
    }
}

impl From<ArbPrecompileError> for PrecompileError {
    fn from(err: ArbPrecompileError) -> Self {
        match err {
            ArbPrecompileError::Revert { .. } => PrecompileError::Other(Cow::Borrowed("revert")),
            ArbPrecompileError::OutOfGas => PrecompileError::OutOfGas,
            ArbPrecompileError::Fatal(source) => PrecompileError::Fatal(source.to_string()),
        }
    }
}
