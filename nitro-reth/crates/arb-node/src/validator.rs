//! Arbitrum engine API payload and attribute validators.

use std::sync::Arc;

use alloy_rpc_types_engine::ExecutionData;
use arb_payload::{ArbBlock, ArbPayloadAttributes};
use arb_primitives::ArbTransactionSigned;
use reth_chainspec::ChainSpec;
use reth_engine_primitives::{EngineApiValidator, PayloadValidator};
use reth_payload_primitives::{
    validate_version_specific_fields, EngineApiMessageVersion, EngineObjectValidationError,
    NewPayloadError, PayloadOrAttributes, PayloadTypes,
};
use reth_primitives_traits::{Block as _, SealedBlock};

/// Arbitrum engine API validator.
///
/// Validates execution payloads and attributes for the engine API,
/// converting raw payloads to sealed blocks with Arbitrum transaction types.
#[derive(Debug, Clone)]
pub struct ArbPayloadValidator {
    chain_spec: Arc<ChainSpec>,
}

impl ArbPayloadValidator {
    /// Create a new validator with the given chain spec.
    pub fn new(chain_spec: Arc<ChainSpec>) -> Self {
        Self { chain_spec }
    }
}

impl<Types> PayloadValidator<Types> for ArbPayloadValidator
where
    Types: PayloadTypes<ExecutionData = ExecutionData>,
{
    type Block = ArbBlock;

    fn convert_payload_to_block(
        &self,
        payload: ExecutionData,
    ) -> Result<SealedBlock<ArbBlock>, NewPayloadError> {
        let ExecutionData { payload, sidecar } = payload;
        let expected_hash = payload.block_hash();

        let sealed_block: SealedBlock<alloy_consensus::Block<ArbTransactionSigned>> =
            payload
                .try_into_block_with_sidecar(&sidecar)
                .map_err(|e| NewPayloadError::Other(e.into()))?
                .seal_slow();

        if expected_hash != sealed_block.hash() {
            return Err(NewPayloadError::Other(
                alloy_rpc_types_engine::PayloadError::BlockHash {
                    execution: sealed_block.hash(),
                    consensus: expected_hash,
                }
                .into(),
            ));
        }

        Ok(sealed_block)
    }
}

impl<Types> EngineApiValidator<Types> for ArbPayloadValidator
where
    Types: PayloadTypes<
        ExecutionData = ExecutionData,
        PayloadAttributes = ArbPayloadAttributes,
    >,
{
    fn validate_version_specific_fields(
        &self,
        version: EngineApiMessageVersion,
        payload_or_attrs: PayloadOrAttributes<
            '_,
            Types::ExecutionData,
            ArbPayloadAttributes,
        >,
    ) -> Result<(), EngineObjectValidationError> {
        validate_version_specific_fields(&*self.chain_spec, version, payload_or_attrs)
    }

    fn ensure_well_formed_attributes(
        &self,
        version: EngineApiMessageVersion,
        attributes: &ArbPayloadAttributes,
    ) -> Result<(), EngineObjectValidationError> {
        validate_version_specific_fields(
            &*self.chain_spec,
            version,
            PayloadOrAttributes::<Types::ExecutionData, ArbPayloadAttributes>::PayloadAttributes(
                attributes,
            ),
        )
    }
}
