//! Arbitrum payload and engine types.
//!
//! Defines the payload attributes, built payload, payload types, and engine
//! types used by the engine API and block construction pipeline.

use std::{marker::PhantomData, sync::Arc};

use alloy_eips::{
    eip4895::{Withdrawal, Withdrawals},
    eip7685::Requests,
};
use alloy_primitives::{Address, Bytes, B256, U256};
use alloy_rpc_types_engine::{
    BlobsBundleV1, BlobsBundleV2, ExecutionData, ExecutionPayload as AlloyExecutionPayload,
    ExecutionPayloadEnvelopeV2, ExecutionPayloadEnvelopeV3, ExecutionPayloadEnvelopeV4,
    ExecutionPayloadEnvelopeV5, ExecutionPayloadEnvelopeV6, ExecutionPayloadFieldV2,
    ExecutionPayloadV1, ExecutionPayloadV3, PayloadAttributes as AlloyPayloadAttributes, PayloadId,
};

use arb_primitives::ArbPrimitives;
use reth_engine_primitives::EngineTypes;
use reth_payload_primitives::{
    BuiltPayload, PayloadAttributes as PayloadAttributesTrait, PayloadBuilderAttributes,
    PayloadTypes,
};
use reth_primitives_traits::{NodePrimitives, SealedBlock};
use serde::{Deserialize, Serialize};

/// Type alias for the Arbitrum block type.
pub type ArbBlock = <ArbPrimitives as NodePrimitives>::Block;

// ── Payload Attributes ────────────────────────────────────────────────────────

/// Arbitrum-specific payload attributes extending the standard engine API.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbPayloadAttributes {
    /// Standard Ethereum payload attributes.
    #[serde(flatten)]
    pub inner: AlloyPayloadAttributes,
    /// Sequencer transactions for this block.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub transactions: Option<Vec<Bytes>>,
    /// Whether to exclude the transaction pool.
    #[serde(default)]
    pub no_tx_pool: bool,
}

impl PayloadAttributesTrait for ArbPayloadAttributes {
    fn timestamp(&self) -> u64 {
        self.inner.timestamp
    }

    fn withdrawals(&self) -> Option<&Vec<Withdrawal>> {
        self.inner.withdrawals.as_ref()
    }

    fn parent_beacon_block_root(&self) -> Option<B256> {
        self.inner.parent_beacon_block_root
    }
}

// ── Payload Builder Attributes ────────────────────────────────────────────────

/// Builder attributes for constructing Arbitrum payloads.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ArbPayloadBuilderAttributes {
    /// Payload identifier.
    pub id: PayloadId,
    /// Parent block hash.
    pub parent: B256,
    /// Target timestamp.
    pub timestamp: u64,
    /// Fee recipient address.
    pub suggested_fee_recipient: Address,
    /// Randomness value.
    pub prev_randao: B256,
    /// Withdrawals to include.
    pub withdrawals: Withdrawals,
    /// Parent beacon block root.
    pub parent_beacon_block_root: Option<B256>,
    /// Whether to exclude the transaction pool.
    pub no_tx_pool: bool,
    /// Forced transactions from the sequencer.
    pub transactions: Vec<Bytes>,
}

impl PayloadBuilderAttributes for ArbPayloadBuilderAttributes {
    type RpcPayloadAttributes = ArbPayloadAttributes;
    type Error = PayloadIdComputeError;

    fn try_new(
        parent: B256,
        attributes: ArbPayloadAttributes,
        _version: u8,
    ) -> Result<Self, Self::Error> {
        let id = arb_payload_id(&parent, &attributes);
        Ok(Self {
            id,
            parent,
            timestamp: attributes.inner.timestamp,
            suggested_fee_recipient: attributes.inner.suggested_fee_recipient,
            prev_randao: attributes.inner.prev_randao,
            withdrawals: attributes.inner.withdrawals.unwrap_or_default().into(),
            parent_beacon_block_root: attributes.inner.parent_beacon_block_root,
            no_tx_pool: attributes.no_tx_pool,
            transactions: attributes.transactions.unwrap_or_default(),
        })
    }

    fn payload_id(&self) -> PayloadId {
        self.id
    }

    fn parent(&self) -> B256 {
        self.parent
    }

    fn timestamp(&self) -> u64 {
        self.timestamp
    }

    fn parent_beacon_block_root(&self) -> Option<B256> {
        self.parent_beacon_block_root
    }

    fn suggested_fee_recipient(&self) -> Address {
        self.suggested_fee_recipient
    }

    fn prev_randao(&self) -> B256 {
        self.prev_randao
    }

    fn withdrawals(&self) -> &Withdrawals {
        &self.withdrawals
    }
}

// ── Payload ID Computation ────────────────────────────────────────────────────

/// Error when computing payload IDs (infallible in practice).
#[derive(Debug, Clone, Copy, thiserror::Error)]
#[error("payload id computation failed")]
pub struct PayloadIdComputeError;

/// Compute a unique payload ID from the parent hash and attributes.
pub fn arb_payload_id(parent: &B256, attributes: &ArbPayloadAttributes) -> PayloadId {
    use alloy_rlp::Encodable;
    use sha2::Digest;

    let mut hasher = sha2::Sha256::new();
    hasher.update(parent.as_slice());
    hasher.update(attributes.inner.timestamp.to_be_bytes());
    hasher.update(attributes.inner.prev_randao.as_slice());
    hasher.update(attributes.inner.suggested_fee_recipient.as_slice());
    if let Some(withdrawals) = &attributes.inner.withdrawals {
        let mut buf = Vec::new();
        withdrawals.encode(&mut buf);
        hasher.update(buf);
    }
    if let Some(root) = attributes.inner.parent_beacon_block_root {
        hasher.update(root);
    }
    // Include Arbitrum-specific fields in the payload ID.
    if attributes.no_tx_pool {
        hasher.update([1u8]);
    }
    if let Some(txs) = &attributes.transactions {
        for tx in txs {
            hasher.update(tx.as_ref());
        }
    }

    let out = hasher.finalize();
    PayloadId::new(out.as_slice()[..8].try_into().expect("sufficient length"))
}

// ── Built Payload ─────────────────────────────────────────────────────────────

/// A built Arbitrum payload ready to be sealed.
#[derive(Debug, Clone)]
pub struct ArbBuiltPayload {
    /// Payload identifier.
    pub id: PayloadId,
    /// The sealed block.
    pub block: Arc<SealedBlock<ArbBlock>>,
    /// Total fees collected.
    pub fees: U256,
    /// Execution requests, if any.
    pub requests: Option<Requests>,
}

impl ArbBuiltPayload {
    /// Create a new built payload.
    pub fn new(id: PayloadId, block: Arc<SealedBlock<ArbBlock>>, fees: U256) -> Self {
        Self {
            id,
            block,
            fees,
            requests: None,
        }
    }

    /// Set execution requests on this payload.
    pub fn with_requests(mut self, requests: Option<Requests>) -> Self {
        self.requests = requests;
        self
    }
}

impl BuiltPayload for ArbBuiltPayload {
    type Primitives = ArbPrimitives;

    fn block(&self) -> &SealedBlock<ArbBlock> {
        &self.block
    }

    fn fees(&self) -> U256 {
        self.fees
    }

    fn requests(&self) -> Option<Requests> {
        self.requests.clone()
    }
}

// ── Conversion Error ──────────────────────────────────────────────────────────

/// Error when converting built payloads to envelope types.
#[derive(Debug, Clone, thiserror::Error)]
#[error("payload conversion failed")]
pub struct ArbPayloadConversionError;

// ── From/TryFrom for execution payload envelopes ──────────────────────────────

// V1
impl From<ArbBuiltPayload> for ExecutionPayloadV1 {
    fn from(value: ArbBuiltPayload) -> Self {
        Self::from_block_unchecked(
            value.block.hash(),
            &Arc::unwrap_or_clone(value.block).into_block(),
        )
    }
}

// V2
impl From<ArbBuiltPayload> for ExecutionPayloadEnvelopeV2 {
    fn from(value: ArbBuiltPayload) -> Self {
        let ArbBuiltPayload { block, fees, .. } = value;
        Self {
            block_value: fees,
            execution_payload: ExecutionPayloadFieldV2::from_block_unchecked(
                block.hash(),
                &Arc::unwrap_or_clone(block).into_block(),
            ),
        }
    }
}

// V3
impl TryFrom<ArbBuiltPayload> for ExecutionPayloadEnvelopeV3 {
    type Error = ArbPayloadConversionError;

    fn try_from(value: ArbBuiltPayload) -> Result<Self, Self::Error> {
        let ArbBuiltPayload { block, fees, .. } = value;
        Ok(Self {
            execution_payload: ExecutionPayloadV3::from_block_unchecked(
                block.hash(),
                &Arc::unwrap_or_clone(block).into_block(),
            ),
            block_value: fees,
            should_override_builder: false,
            blobs_bundle: BlobsBundleV1::empty(),
        })
    }
}

// V4
impl TryFrom<ArbBuiltPayload> for ExecutionPayloadEnvelopeV4 {
    type Error = ArbPayloadConversionError;

    fn try_from(value: ArbBuiltPayload) -> Result<Self, Self::Error> {
        let requests = value.requests.clone().unwrap_or_default();
        let v3: ExecutionPayloadEnvelopeV3 = value.try_into()?;
        Ok(Self {
            execution_requests: requests,
            envelope_inner: v3,
        })
    }
}

// V5
impl TryFrom<ArbBuiltPayload> for ExecutionPayloadEnvelopeV5 {
    type Error = ArbPayloadConversionError;

    fn try_from(value: ArbBuiltPayload) -> Result<Self, Self::Error> {
        let ArbBuiltPayload {
            block,
            fees,
            requests,
            ..
        } = value;
        Ok(Self {
            execution_payload: ExecutionPayloadV3::from_block_unchecked(
                block.hash(),
                &Arc::unwrap_or_clone(block).into_block(),
            ),
            block_value: fees,
            should_override_builder: false,
            blobs_bundle: BlobsBundleV2::empty(),
            execution_requests: requests.unwrap_or_default(),
        })
    }
}

// V6
impl TryFrom<ArbBuiltPayload> for ExecutionPayloadEnvelopeV6 {
    type Error = ArbPayloadConversionError;

    fn try_from(_value: ArbBuiltPayload) -> Result<Self, Self::Error> {
        Err(ArbPayloadConversionError)
    }
}

// ── Payload Types ─────────────────────────────────────────────────────────────

/// Payload types for the Arbitrum engine.
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
#[non_exhaustive]
pub struct ArbPayloadTypes;

impl PayloadTypes for ArbPayloadTypes {
    type ExecutionData = ExecutionData;
    type BuiltPayload = ArbBuiltPayload;
    type PayloadAttributes = ArbPayloadAttributes;
    type PayloadBuilderAttributes = ArbPayloadBuilderAttributes;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> Self::ExecutionData {
        let (payload, sidecar) =
            AlloyExecutionPayload::from_block_unchecked(block.hash(), &block.into_block());
        ExecutionData { payload, sidecar }
    }
}

// ── Engine Types ──────────────────────────────────────────────────────────────

/// Engine types for the Arbitrum consensus engine.
#[derive(Debug, Default, Clone, Serialize, Deserialize)]
#[non_exhaustive]
pub struct ArbEngineTypes<T: PayloadTypes = ArbPayloadTypes> {
    _marker: PhantomData<T>,
}

impl<T: PayloadTypes<ExecutionData = ExecutionData>> PayloadTypes for ArbEngineTypes<T>
where
    T::BuiltPayload: BuiltPayload<Primitives: NodePrimitives<Block = ArbBlock>>,
{
    type ExecutionData = T::ExecutionData;
    type BuiltPayload = T::BuiltPayload;
    type PayloadAttributes = T::PayloadAttributes;
    type PayloadBuilderAttributes = T::PayloadBuilderAttributes;

    fn block_to_payload(
        block: SealedBlock<
            <<Self::BuiltPayload as BuiltPayload>::Primitives as NodePrimitives>::Block,
        >,
    ) -> Self::ExecutionData {
        T::block_to_payload(block)
    }
}

impl<T> EngineTypes for ArbEngineTypes<T>
where
    T: PayloadTypes<ExecutionData = ExecutionData>,
    T::BuiltPayload: BuiltPayload<Primitives: NodePrimitives<Block = ArbBlock>>
        + TryInto<ExecutionPayloadV1>
        + TryInto<ExecutionPayloadEnvelopeV2>
        + TryInto<ExecutionPayloadEnvelopeV3>
        + TryInto<ExecutionPayloadEnvelopeV4>
        + TryInto<ExecutionPayloadEnvelopeV5>
        + TryInto<ExecutionPayloadEnvelopeV6>,
{
    type ExecutionPayloadEnvelopeV1 = ExecutionPayloadV1;
    type ExecutionPayloadEnvelopeV2 = ExecutionPayloadEnvelopeV2;
    type ExecutionPayloadEnvelopeV3 = ExecutionPayloadEnvelopeV3;
    type ExecutionPayloadEnvelopeV4 = ExecutionPayloadEnvelopeV4;
    type ExecutionPayloadEnvelopeV5 = ExecutionPayloadEnvelopeV5;
    type ExecutionPayloadEnvelopeV6 = ExecutionPayloadEnvelopeV6;
}
