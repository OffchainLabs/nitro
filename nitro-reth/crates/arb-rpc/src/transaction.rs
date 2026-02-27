//! Arbitrum transaction request and conversion types.

use alloy_consensus::{error::ValueError, SignableTransaction};
use alloy_primitives::Signature;
use alloy_rpc_types_eth::request::TransactionRequest;
use arb_primitives::ArbTransactionSigned;
use reth_rpc_convert::{SignTxRequestError, SignableTxRequest, TryIntoSimTx};
use serde::{Deserialize, Serialize};

/// Arbitrum transaction request wrapping the standard Ethereum transaction request.
///
/// This newtype allows implementing Arbitrum-specific RPC traits while
/// delegating serialization and most behavior to the inner type.
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(transparent)]
pub struct ArbTransactionRequest(pub TransactionRequest);

impl AsRef<TransactionRequest> for ArbTransactionRequest {
    fn as_ref(&self) -> &TransactionRequest {
        &self.0
    }
}

impl AsMut<TransactionRequest> for ArbTransactionRequest {
    fn as_mut(&mut self) -> &mut TransactionRequest {
        &mut self.0
    }
}

impl SignableTxRequest<ArbTransactionSigned> for ArbTransactionRequest {
    async fn try_build_and_sign(
        self,
        signer: impl alloy_network::TxSigner<Signature> + Send,
    ) -> Result<ArbTransactionSigned, SignTxRequestError> {
        // Build a standard typed transaction, sign it, then wrap as Arbitrum.
        let mut tx = self
            .0
            .build_typed_tx()
            .map_err(|_| SignTxRequestError::InvalidTransactionRequest)?;
        let signature = signer.sign_transaction(&mut tx).await?;
        let signed = tx.into_signed(signature);
        Ok(ArbTransactionSigned::from_envelope(signed.into()))
    }
}

impl TryIntoSimTx<ArbTransactionSigned> for ArbTransactionRequest {
    fn try_into_sim_tx(self) -> Result<ArbTransactionSigned, ValueError<Self>> {
        Err(ValueError::new(
            self,
            "simulate_v1 not yet supported",
        ))
    }
}
