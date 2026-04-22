//! Arbitrum transaction request and conversion types.

use alloy_consensus::{error::ValueError, SignableTransaction};
use alloy_evm::rpc::TryIntoTxEnv;
use alloy_primitives::Signature;
use alloy_rpc_types_eth::request::TransactionRequest;
use arb_primitives::ArbTransactionSigned;
use reth_rpc_convert::{SignTxRequestError, SignableTxRequest, TryIntoSimTx};
use serde::{Deserialize, Serialize};

/// Arbitrum transaction request wrapping the standard Ethereum transaction request.
///
/// This newtype allows implementing Arbitrum-specific RPC traits while
/// delegating serialization and most behavior to the inner type.
#[derive(Clone, Debug, Default, Serialize, Deserialize)]
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

impl From<TransactionRequest> for ArbTransactionRequest {
    fn from(req: TransactionRequest) -> Self {
        Self(req)
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
        // Build the typed simulation tx via alloy's reference impl (fills in
        // defaults for missing fields and wraps with a placeholder signature),
        // then wrap into the Arbitrum envelope.
        match TransactionRequest::build_typed_simulate_transaction(self.0.clone()) {
            Ok(envelope) => Ok(ArbTransactionSigned::from_envelope(envelope)),
            Err(err) => Err(ValueError::new(self, err.to_string())),
        }
    }
}

impl<Block: alloy_evm::env::BlockEnvironment> TryIntoTxEnv<arb_evm::ArbTransaction, Block>
    for ArbTransactionRequest
{
    type Err = alloy_evm::rpc::EthTxEnvError;

    fn try_into_tx_env<Spec>(
        self,
        evm_env: &alloy_evm::EvmEnv<Spec, Block>,
    ) -> Result<arb_evm::ArbTransaction, Self::Err> {
        let tx_env = self.0.try_into_tx_env(evm_env)?;
        Ok(arb_evm::ArbTransaction(tx_env))
    }
}
